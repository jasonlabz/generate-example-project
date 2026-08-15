package wire

import "github.com/danielgtaylor/huma/v2"

// Module is the Huma registration descriptor returned by a module Wire.
//
// Router owns the stable service/api/v1 group hierarchy. A module receives a
// Huma API for the scope it registers in and may create nested Huma groups for
// its own resource prefix or middleware.
type Module struct {
	Name         string
	RegisterRoot func(huma.API)
	RegisterBase func(huma.API)
	RegisterV1   func(huma.API)
}

// MountRoot registers routes that live outside the service API group.
func (m Module) MountRoot(api huma.API) {
	if m.RegisterRoot != nil {
		m.RegisterRoot(api)
	}
}

// MountBase registers module routes below the service API group.
func (m Module) MountBase(api huma.API) {
	if m.RegisterBase != nil {
		m.RegisterBase(api)
	}
}

// MountV1 registers module routes below the versioned API group.
func (m Module) MountV1(api huma.API) {
	if m.RegisterV1 != nil {
		m.RegisterV1(api)
	}
}
