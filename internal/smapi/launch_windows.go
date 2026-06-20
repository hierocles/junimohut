package smapi

import "golang.org/x/sys/windows"

// launchProcess starts SMAPI via ShellExecuteW so Windows wires up the new
// console's stdin/stdout/stderr handles properly — exec.Command always sets
// STARTF_USESTDHANDLES with NUL handles, which silences all console output.
func launchProcess(path, dir string) error {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	dirPtr, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, nil, pathPtr, nil, dirPtr, windows.SW_SHOWNORMAL)
}
