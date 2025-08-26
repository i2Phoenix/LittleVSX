package marketplace

// MarketplaceProvider defines the interface for different marketplace implementations
type MarketplaceProvider interface {
	GetExtensionInfo(marketplaceURL string) (*ExtensionInfo, error)
	GetExtensionInfoByID(extensionID string) (*ExtensionInfo, error)
	DownloadExtension(info *ExtensionInfo, targetDir string) (*DownloadResult, error)
	GetName() string
}

// MarketplaceType represents the type of marketplace
type MarketplaceType string

const (
	MarketplaceTypeMicrosoft MarketplaceType = "microsoft"
	MarketplaceTypeOpenVSX   MarketplaceType = "open-vsx"
)
