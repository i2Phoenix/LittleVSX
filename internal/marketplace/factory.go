package marketplace

import (
	"fmt"
)

// Factory creates marketplace providers based on type
type Factory struct{}

// NewFactory creates a new marketplace factory
func NewFactory() *Factory {
	return &Factory{}
}

// CreateByType creates a marketplace provider by type
func (f *Factory) CreateByType(marketplaceType MarketplaceType) (MarketplaceProvider, error) {
	switch marketplaceType {
	case MarketplaceTypeMicrosoft:
		return NewMicrosoft(), nil
	case MarketplaceTypeOpenVSX:
		return NewOpenVSX(), nil
	default:
		return nil, fmt.Errorf("unknown marketplace type: %s", marketplaceType)
	}
}
