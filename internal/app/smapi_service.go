package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"junimohut/internal/config"
	"junimohut/internal/moddataset"
	"junimohut/internal/mods"
	"junimohut/internal/nexus"
	"junimohut/internal/smapi"
)

type SMAPIService struct { core *Core }
func NewSMAPIService(core *Core) *SMAPIService { return &SMAPIService{core: core} }

func (s *SMAPIService) LaunchSMAPI() error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	settings := s.core.Store.Get()
	if err := s.core.Catalog.Refresh(s.core.Ctx()); err != nil {
		return err
	}
	if settings.ProfileSpecificConfigs {
		if err := s.core.ConfigMgr.SaveConfigs(settings.ModsRoot, modUniqueIDMapFromEnabled(s.core, s.core.Profiles.EnabledMods())); err != nil {
			return err
		}
	}
	launcher := smapi.NewLauncher(settings.GamePath, settings.SMAPIPath)
	return launcher.Launch()
}

func (s *SMAPIService) GetSMAPIVersion() string {
	if err := s.core.RequireStarted(); err != nil {
		return ""
	}
	settings := s.core.Store.Get()
	return smapi.NewLauncher(settings.GamePath, settings.SMAPIPath).Version()
}

func (s *SMAPIService) CheckSMAPIUpdate() (smapi.UpdateInfo, error) {
	if err := s.core.RequireStarted(); err != nil {
		return smapi.UpdateInfo{}, err
	}
	return smapi.CheckSMAPIUpdate(s.core.SMAPIVersion())
}

func (s *SMAPIService) InstallSMAPI() error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	info, _ := smapi.CheckSMAPIUpdate(s.core.SMAPIVersion())
	return smapi.InstallSMAPI(info.DownloadURL, s.core.Store.Get().GamePath)
}

func (s *SMAPIService) CheckModUpdates() ([]smapi.ModUpdateResult, error) {
	if err := s.core.RequireStarted(); err != nil {
		return nil, err
	}
	settings := s.core.Store.Get()
	now := time.Now()
	if smapi.UpdateCheckRateLimited(settings.LastUpdateCheck, now) {
		retryAfter := smapi.UpdateCheckRetryAfter(settings.LastUpdateCheck, now)
		return nil, fmt.Errorf("%w — try again in %s", smapi.ErrUpdateCheckRateLimited, smapi.FormatUpdateCheckRetryAfter(retryAfter))
	}
	list := s.core.Catalog.CopyMods()

	var requests []smapi.ModUpdateRequest
	seenNexus := map[int]bool{}
	seenUnique := map[string]bool{}
	for _, m := range list {
		updateKeys := append([]string(nil), m.Manifest.UpdateKeys...)
		if len(updateKeys) == 0 && s.core.ModDataset != nil {
			updateKeys = moddataset.NexusUpdateKeysFromIndex(s.core.ModDataset, m.Manifest.UniqueID)
		}
		if len(updateKeys) == 0 {
			continue
		}
		nexusID := nexus.ModIDFromUpdateKeys(updateKeys)
		if nexusID > 0 {
			if seenNexus[nexusID] {
				continue
			}
			seenNexus[nexusID] = true
			rep := m
			if len(m.BundleChildren) > 0 {
				if childRep, ok := mods.PickNexusGroupRepresentative(m.BundleChildren, nexusID); ok {
					rep = childRep
				}
			} else if groupRep, ok := mods.PickNexusGroupRepresentative(list, nexusID); ok {
				rep = groupRep
			}
			if seenUnique[rep.Manifest.UniqueID] {
				continue
			}
			seenUnique[rep.Manifest.UniqueID] = true
			keys := rep.Manifest.UpdateKeys
			if len(keys) == 0 {
				keys = updateKeys
			}
			requests = append(requests, smapi.ModUpdateRequest{
				UniqueID:   rep.Manifest.UniqueID,
				Version:    rep.Manifest.Version,
				UpdateKeys: keys,
			})
			continue
		}
		if seenUnique[m.Manifest.UniqueID] {
			continue
		}
		seenUnique[m.Manifest.UniqueID] = true
		requests = append(requests, smapi.ModUpdateRequest{
			UniqueID:   m.Manifest.UniqueID,
			Version:    m.Manifest.Version,
			UpdateKeys: updateKeys,
		})
	}
	results, err := smapi.CheckModUpdates(requests, s.core.SMAPIVersion())
	if err != nil {
		return nil, err
	}
	_ = s.core.Store.Update(func(st *config.Settings) {
		st.LastUpdateCheck = time.Now().Unix()
	})
	resultMap := map[string]smapi.ModUpdateResult{}
	for _, r := range results {
		resultMap[r.UniqueID] = r
	}
	nexusResults := map[int]smapi.ModUpdateResult{}
	for _, req := range requests {
		if r, ok := resultMap[req.UniqueID]; ok {
			if id := nexus.ModIDFromUpdateKeys(req.UpdateKeys); id > 0 {
				nexusResults[id] = r
			}
		}
	}
	s.core.Catalog.mu.Lock()
	for i, m := range s.core.Catalog.mods {
		var r smapi.ModUpdateResult
		var ok bool
		nexusID := m.ResolvedNexusModID
		if nexusID == 0 {
			nexusID = nexus.ModIDFromUpdateKeys(m.Manifest.UpdateKeys)
		}
		if nexusID > 0 {
			r, ok = nexusResults[nexusID]
		} else {
			r, ok = resultMap[m.Manifest.UniqueID]
		}
		if !ok {
			continue
		}
		s.core.Catalog.mods[i].UpdateStatus = mods.UpdateStatus{
			State:                mods.NormalizeUpdateState(r.Status),
			LatestVersion:        r.LatestVersion,
			ModPageURL:           r.ModPageURL,
			Message:              r.Message,
			CompatibilityStatus:  r.CompatibilityStatus,
			CompatibilitySummary: r.CompatibilitySummary,
		}
	}
	mods.PropagateNexusUpdateStatus(s.core.Catalog.mods)
	mods.ApplyIgnoredUpdates(s.core.Catalog.mods, settings.IgnoredModUpdates)
	mods.StripBundleChildUpdateStatus(s.core.Catalog.mods)
	s.core.Catalog.mu.Unlock()
	s.core.Catalog.SyncModUpdateCache()
	return results, nil
}

func (s *SMAPIService) SetModUpdateIgnored(modID string, ignored bool) error {
	if err := s.core.RequireStarted(); err != nil {
		return err
	}
	target, ok := resolveUpdateMod(s.core, modID)
	if !ok {
		return fmt.Errorf("mod not found")
	}
	nexusID := nexus.ModIDFromUpdateKeys(target.Manifest.UpdateKeys)
	if nexusID == 0 {
		return fmt.Errorf("mod has no Nexus update key")
	}
	latest := strings.TrimSpace(target.UpdateStatus.LatestVersion)
	if ignored && latest == "" {
		return fmt.Errorf("no available update to ignore")
	}
	key := strconv.Itoa(nexusID)
	if err := s.core.Store.Update(func(st *config.Settings) {
		if st.IgnoredModUpdates == nil {
			st.IgnoredModUpdates = map[string]string{}
		}
		if ignored {
			st.IgnoredModUpdates[key] = latest
		} else {
			delete(st.IgnoredModUpdates, key)
		}
	}); err != nil {
		return err
	}
	s.core.Catalog.mu.Lock()
	mods.ApplyIgnoredUpdates(s.core.Catalog.mods, s.core.Store.Get().IgnoredModUpdates)
	s.core.Catalog.mu.Unlock()
	s.core.Catalog.SyncModUpdateCache()
	return nil
}
