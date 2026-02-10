package capture

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <CoreGraphics/CoreGraphics.h>
#include <dlfcn.h>
#include <stdlib.h>

typedef struct {
    void*  data;
    size_t size;
    int    width;
    int    height;
    size_t bytesPerRow;
} FrameData;

// CGWindowListCreateImage is unavailable in the macOS 15 SDK headers but still
// present in the CoreGraphics dylib. Load it dynamically.
typedef CGImageRef (*CGWindowListCreateImageFunc)(
    CGRect screenBounds,
    uint32_t listOption,
    uint32_t windowID,
    uint32_t imageOption
);

static CGWindowListCreateImageFunc getCGWindowListCreateImage(void) {
    static CGWindowListCreateImageFunc fn = NULL;
    if (!fn) {
        fn = (CGWindowListCreateImageFunc)dlsym(RTLD_DEFAULT, "CGWindowListCreateImage");
    }
    return fn;
}

FrameData captureDisplay(CGDirectDisplayID displayID) {
    FrameData result = {0};

    CGWindowListCreateImageFunc fn = getCGWindowListCreateImage();
    if (!fn) {
        return result;
    }

    CGRect bounds = CGDisplayBounds(displayID);
    // kCGWindowListOptionOnScreenOnly = 1, kCGNullWindowID = 0, kCGWindowImageDefault = 0
    CGImageRef image = fn(bounds, 1, 0, 0);
    if (!image) {
        return result;
    }

    result.width  = (int)CGImageGetWidth(image);
    result.height = (int)CGImageGetHeight(image);

    result.bytesPerRow = result.width * 4;
    result.size        = result.bytesPerRow * result.height;
    result.data        = malloc(result.size);
    if (!result.data) {
        CGImageRelease(image);
        result.size = 0;
        return result;
    }

    CGColorSpaceRef cs = CGColorSpaceCreateDeviceRGB();
    CGContextRef ctx = CGBitmapContextCreate(
        result.data,
        result.width,
        result.height,
        8,
        result.bytesPerRow,
        cs,
        kCGImageAlphaPremultipliedLast
    );
    CGContextDrawImage(ctx, CGRectMake(0, 0, result.width, result.height), image);
    CGContextRelease(ctx);
    CGColorSpaceRelease(cs);
    CGImageRelease(image);

    return result;
}

void freeFrameData(void* data) {
    free(data);
}
*/
import "C"

import (
	"fmt"
	"image"
	"time"
	"unsafe"
)

// CGCapturer implements Capturer using CoreGraphics.
type CGCapturer struct {
	displayID C.CGDirectDisplayID
	fps       int
	frameCh   chan *Frame
	stopCh    chan struct{}
	running   bool
}

// NewCGCapturer creates a screen capturer for the given display at the given FPS.
func NewCGCapturer(displayIndex int, fps int) (*CGCapturer, error) {
	if fps <= 0 || fps > 60 {
		return nil, fmt.Errorf("fps must be 1-60, got %d", fps)
	}

	var displayID C.CGDirectDisplayID
	if displayIndex == 0 {
		displayID = C.CGMainDisplayID()
	} else {
		var displays [16]C.CGDirectDisplayID
		var count C.uint32_t
		C.CGGetActiveDisplayList(16, &displays[0], &count)
		if displayIndex >= int(count) {
			return nil, fmt.Errorf("display index %d out of range (have %d displays)", displayIndex, count)
		}
		displayID = displays[displayIndex]
	}

	return &CGCapturer{
		displayID: displayID,
		fps:       fps,
		frameCh:   make(chan *Frame, 2),
		stopCh:    make(chan struct{}),
	}, nil
}

func (c *CGCapturer) Start() error {
	if c.running {
		return fmt.Errorf("already running")
	}
	c.running = true
	go c.loop()
	return nil
}

func (c *CGCapturer) Stop() {
	if !c.running {
		return
	}
	c.running = false
	close(c.stopCh)
}

func (c *CGCapturer) Frames() <-chan *Frame {
	return c.frameCh
}

func (c *CGCapturer) loop() {
	ticker := time.NewTicker(time.Second / time.Duration(c.fps))
	defer ticker.Stop()
	defer close(c.frameCh)

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			f := c.capture()
			if f == nil {
				continue
			}
			select {
			case c.frameCh <- f:
			default:
			}
		}
	}
}

func (c *CGCapturer) capture() *Frame {
	fd := C.captureDisplay(c.displayID)
	if fd.data == nil {
		return nil
	}
	defer C.freeFrameData(fd.data)

	w := int(fd.width)
	h := int(fd.height)
	byteLen := int(fd.size)

	pix := make([]byte, byteLen)
	copy(pix, unsafe.Slice((*byte)(fd.data), byteLen))

	img := &image.RGBA{
		Pix:    pix,
		Stride: w * 4,
		Rect:   image.Rect(0, 0, w, h),
	}

	return &Frame{
		Image:     img,
		Width:     w,
		Height:    h,
		Timestamp: time.Now(),
	}
}
