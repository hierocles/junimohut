package main

// SelfUpdate checks for application updates (placeholder for release channel).
func (a *App) CheckAppUpdate() (string, error) {
	return "0.1.0", nil
}

// RegisterNXMProtocol registers nxm:// to launch this app (Windows, current user).
func (a *App) RegisterNXMProtocol() error {
	return registerNXMProtocol()
}
