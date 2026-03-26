package tui

import "github.com/losinggeneration/tui/backend"

type runtimeHost interface {
	Backend() backend.Backend
	Enable() (Size, error)
	Restore() error
	ReadEvent() Event
	Size() Size
	InputCapabilities() backend.InputCapabilities
	SetInputFeatures(f backend.InputFeatures) error
	ClipboardWrite(text string) bool
	CanClipboardReadAsync() bool
	ClipboardReadRequest() bool
}

type appHost struct {
	raw backend.Backend
}

func newAppHost(b backend.Backend) *appHost {
	return &appHost{raw: b}
}

func (h *appHost) Backend() backend.Backend {
	return h.raw
}

func (h *appHost) Enable() (Size, error) {
	return h.raw.Enable()
}

func (h *appHost) Restore() error {
	return h.raw.Restore()
}

func (h *appHost) ReadEvent() Event {
	return h.raw.ReadEvent()
}

func (h *appHost) Size() Size {
	return h.raw.Size()
}

func (h *appHost) InputCapabilities() backend.InputCapabilities {
	if cr, ok := h.raw.(backend.CapabilityReporter); ok {
		return cr.InputCapabilities()
	}

	return backend.InputCapabilities{}
}

func (h *appHost) SetInputFeatures(f backend.InputFeatures) error {
	if fe, ok := h.raw.(backend.InputFeatureEnabler); ok {
		return fe.SetInputFeatures(f)
	}

	return nil
}

func (h *appHost) ClipboardWrite(text string) bool {
	cb, ok := h.raw.(backend.ClipboardBackend)
	if !ok {
		return false
	}

	_ = cb.ClipboardWrite(text)
	return true
}

func (h *appHost) CanClipboardReadAsync() bool {
	_, ok := h.raw.(backend.ClipboardAsyncReader)
	return ok
}

func (h *appHost) ClipboardReadRequest() bool {
	ar, ok := h.raw.(backend.ClipboardAsyncReader)
	if !ok {
		return false
	}

	_ = ar.ClipboardReadRequest()
	return true
}
