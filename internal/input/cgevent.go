package input

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>

void moveMouse(double x, double y) {
    CGEventRef event = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved,
        CGPointMake(x, y), kCGMouseButtonLeft);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}

void mouseDown(double x, double y, int button) {
    CGEventType type;
    CGMouseButton btn;
    switch (button) {
        case 1:  type = kCGEventRightMouseDown;  btn = kCGMouseButtonRight;  break;
        case 2:  type = kCGEventOtherMouseDown;  btn = kCGMouseButtonCenter; break;
        default: type = kCGEventLeftMouseDown;   btn = kCGMouseButtonLeft;   break;
    }
    CGEventRef event = CGEventCreateMouseEvent(NULL, type, CGPointMake(x, y), btn);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}

void mouseUp(double x, double y, int button) {
    CGEventType type;
    CGMouseButton btn;
    switch (button) {
        case 1:  type = kCGEventRightMouseUp;  btn = kCGMouseButtonRight;  break;
        case 2:  type = kCGEventOtherMouseUp;  btn = kCGMouseButtonCenter; break;
        default: type = kCGEventLeftMouseUp;   btn = kCGMouseButtonLeft;   break;
    }
    CGEventRef event = CGEventCreateMouseEvent(NULL, type, CGPointMake(x, y), btn);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}

void mouseScroll(int dx, int dy) {
    CGEventRef event = CGEventCreateScrollWheelEvent(NULL,
        kCGScrollEventUnitPixel, 2, dy, dx);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}

void keyDown(CGKeyCode keyCode, CGEventFlags flags) {
    CGEventRef event = CGEventCreateKeyboardEvent(NULL, keyCode, true);
    if (flags) {
        CGEventSetFlags(event, flags);
    }
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}

void keyUp(CGKeyCode keyCode, CGEventFlags flags) {
    CGEventRef event = CGEventCreateKeyboardEvent(NULL, keyCode, false);
    if (flags) {
        CGEventSetFlags(event, flags);
    }
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}
*/
import "C"

// CGEventInjector injects input via CoreGraphics CGEvent APIs.
type CGEventInjector struct{}

func NewCGEventInjector() *CGEventInjector {
	return &CGEventInjector{}
}

func (inj *CGEventInjector) Inject(e *InputEvent) error {
	flags := modifiersToFlags(e.Modifiers)

	switch e.Type {
	case EventMouseMove:
		C.moveMouse(C.double(e.X), C.double(e.Y))
	case EventMouseDown:
		C.mouseDown(C.double(e.X), C.double(e.Y), C.int(e.Button))
	case EventMouseUp:
		C.mouseUp(C.double(e.X), C.double(e.Y), C.int(e.Button))
	case EventMouseScroll:
		C.mouseScroll(C.int(e.ScrollDX), C.int(e.ScrollDY))
	case EventKeyDown:
		C.keyDown(C.CGKeyCode(e.KeyCode), C.CGEventFlags(flags))
	case EventKeyUp:
		C.keyUp(C.CGKeyCode(e.KeyCode), C.CGEventFlags(flags))
	}
	return nil
}

func modifiersToFlags(m uint8) uint64 {
	var flags uint64
	if m&1 != 0 {
		flags |= 0x00020000 // kCGEventFlagMaskShift
	}
	if m&2 != 0 {
		flags |= 0x00040000 // kCGEventFlagMaskControl
	}
	if m&4 != 0 {
		flags |= 0x00080000 // kCGEventFlagMaskAlternate
	}
	if m&8 != 0 {
		flags |= 0x00100000 // kCGEventFlagMaskCommand
	}
	return flags
}
