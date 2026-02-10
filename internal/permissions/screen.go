package permissions

/*
#cgo LDFLAGS: -framework CoreGraphics
#include <CoreGraphics/CoreGraphics.h>

// CGPreflightScreenCaptureAccess and CGRequestScreenCaptureAccess
// are available since macOS 10.15.
int hasScreenRecordingPermission() {
    return CGPreflightScreenCaptureAccess();
}

int requestScreenRecordingPermission() {
    return CGRequestScreenCaptureAccess();
}
*/
import "C"

// HasScreenRecording returns true if the app has Screen Recording permission.
func HasScreenRecording() bool {
	return C.hasScreenRecordingPermission() != 0
}

// RequestScreenRecording prompts the user for Screen Recording permission.
// Returns true if already granted. If not granted, macOS shows a dialog
// and the user must restart the app after granting.
func RequestScreenRecording() bool {
	return C.requestScreenRecordingPermission() != 0
}
