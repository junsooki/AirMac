package permissions

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>

int hasAccessibilityPermission() {
    // kAXTrustedCheckOptionPrompt = false → just check, don't prompt
    CFMutableDictionaryRef opts = CFDictionaryCreateMutable(NULL, 0, NULL, NULL);
    CFDictionarySetValue(opts, kAXTrustedCheckOptionPrompt, kCFBooleanFalse);
    Boolean trusted = AXIsProcessTrustedWithOptions(opts);
    CFRelease(opts);
    return trusted ? 1 : 0;
}

int requestAccessibilityPermission() {
    // kAXTrustedCheckOptionPrompt = true → prompt the user
    CFMutableDictionaryRef opts = CFDictionaryCreateMutable(NULL, 0, NULL, NULL);
    CFDictionarySetValue(opts, kAXTrustedCheckOptionPrompt, kCFBooleanTrue);
    Boolean trusted = AXIsProcessTrustedWithOptions(opts);
    CFRelease(opts);
    return trusted ? 1 : 0;
}
*/
import "C"

// HasAccessibility returns true if the app has Accessibility permission.
func HasAccessibility() bool {
	return C.hasAccessibilityPermission() != 0
}

// RequestAccessibility prompts the user for Accessibility permission.
// Returns true if already granted. Otherwise macOS shows System Settings.
func RequestAccessibility() bool {
	return C.requestAccessibilityPermission() != 0
}
