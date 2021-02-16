package trust

type ReadOnly interface {
	GetSignedMetadata(string) (*trustRepo, error)
}

type Writable interface {
	InitNotaryRepoWithSigners() error
	SignImage() error

	GetPassphrase(id string) (string, error)
	CreateRootKey() error
	WriteKey(keyId string, key []byte) error
	ReadRootKey() (string, []byte, error)
	ReadTargetKey() (string, []byte, error)
	ClearDir() error
}

type NotaryRepository interface {
	ReadOnly
	Writable
}

// trustTagKey represents a unique signed tag and hex-encoded hash pair
type trustTagKey struct {
	SignedTag string
	Digest    string
}

// trustTagRow encodes all human-consumable information for a signed tag, including signers
type trustTagRow struct {
	trustTagKey
	Signers []string
}

// trustRepo represents consumable information about a trusted repository
type trustRepo struct {
	Name               string
	SignedTags         []trustTagRow
	Signers            []trustSigner
	AdministrativeKeys []trustSigner
}

// trustSigner represents a trusted signer in a trusted repository
// a signer is defined by a name and list of trustKeys
type trustSigner struct {
	Name string     `json:",omitempty"`
	Keys []trustKey `json:",omitempty"`
}

// trustKey contains information about trusted keys
type trustKey struct {
	ID string `json:",omitempty"`
}
