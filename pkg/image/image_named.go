package image

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/docker/distribution/reference"
)

const (
	// DefaultTag defines the default tag used when performing images related actions and no tag or digest is specified
	DefaultTag = "latest"
	// DefaultHostname is the default built-in hostname
	DefaultHostname = "docker.io"
	// LegacyDefaultHostname is automatically converted to DefaultHostname
	LegacyDefaultHostname = "index.docker.io"
	// DefaultRepoPrefix is the prefix used for default repositories in default host
	DefaultRepoPrefix = "library/"
)

// Named is an object with a full name
type Named interface {
	// Name returns normalized repository name, like "ubuntu".
	Name() string
	// String returns full reference, like "ubuntu@sha256:abcdef..."
	String() string
	// FullName returns full repository name with hostname, like "docker.io/library/ubuntu"
	FullName() string
	// Hostname returns hostname for the reference, like "docker.io"
	Hostname() string
	// RemoteName returns the repository component of the full name, like "library/ubuntu"
	RemoteName() string
}

// NamedTagged is an object including a name and tag.
type NamedTagged interface {
	Named
	Tag() string
}

// ParseNamed parses s and returns a syntactically valid reference implementing
// the Named interface. The reference must have a name, otherwise an error is
// returned.
// If an error was encountered it is returned, along with a nil Reference.
func ParseNamed(s string) (Named, error) {
	named, err := reference.ParseNamed(s)
	if err != nil {
		return nil, fmt.Errorf("error parsing reference: %q is not a valid repository/tag", s)
	}
	r, err := WithName(named.Name())
	if err != nil {
		return nil, err
	}
	if tagged, isTagged := named.(reference.NamedTagged); isTagged {
		return WithTag(r, tagged.Tag())
	}
	return r, nil
}

// WithName returns a named object representing the given string. If the input
// is invalid ErrReferenceInvalidFormat will be returned.
func WithName(name string) (Named, error) {
	name, err := normalize(name)
	if err != nil {
		return nil, err
	}
	if err := validateName(name); err != nil {
		return nil, err
	}
	r, err := reference.WithName(name)
	if err != nil {
		return nil, err
	}
	return &namedRef{r}, nil
}

// WithTag combines the name from "name" and the tag from "tag" to form a
// reference incorporating both the name and the tag.
func WithTag(name Named, tag string) (NamedTagged, error) {
	r, err := reference.WithTag(name, tag)
	if err != nil {
		return nil, err
	}
	return &taggedRef{namedRef{r}}, nil
}

type namedRef struct {
	reference.Named
}
type taggedRef struct {
	namedRef
}

func (r *namedRef) FullName() string {
	hostname, remoteName := splitHostname(r.Name())
	return hostname + "/" + remoteName
}
func (r *namedRef) Hostname() string {
	hostname, _ := splitHostname(r.Name())
	return hostname
}
func (r *namedRef) RemoteName() string {
	_, remoteName := splitHostname(r.Name())
	return remoteName
}
func (r *taggedRef) Tag() string {
	return r.namedRef.Named.(reference.NamedTagged).Tag()
}

// WithDefaultTag adds a default tag to a reference if it only has a repo name.
func WithDefaultTag(ref Named) Named {
	if IsNameOnly(ref) {
		ref, _ = WithTag(ref, DefaultTag)
	}
	return ref
}

// IsNameOnly returns true if reference only contains a repo name.
func IsNameOnly(ref Named) bool {
	if _, ok := ref.(NamedTagged); ok {
		return false
	}
	return true
}

// splitHostname splits a repository name to hostname and remotename string.
// If no valid hostname is found, the default hostname is used. Repository name
// needs to be already validated before.
func splitHostname(name string) (hostname, remoteName string) {
	i := strings.IndexRune(name, '/')
	if i == -1 || (!strings.ContainsAny(name[:i], ".:") && name[:i] != "localhost") {
		hostname, remoteName = DefaultHostname, name
	} else {
		hostname, remoteName = name[:i], name[i+1:]
	}
	if hostname == LegacyDefaultHostname {
		hostname = DefaultHostname
	}
	if hostname == DefaultHostname && !strings.ContainsRune(remoteName, '/') {
		remoteName = DefaultRepoPrefix + remoteName
	}
	return
}

// normalize returns a repository name in its normalized form, meaning it
// will not contain default hostname nor library/ prefix for official images.
func normalize(name string) (string, error) {
	host, remoteName := splitHostname(name)
	if strings.ToLower(remoteName) != remoteName {
		return "", fmt.Errorf("invalid reference format: repository name must be lowercase")
	}
	if host == DefaultHostname {
		if strings.HasPrefix(remoteName, DefaultRepoPrefix) {
			return strings.TrimPrefix(remoteName, DefaultRepoPrefix), nil
		}
		return remoteName, nil
	}
	return name, nil
}

func validateName(name string) error {
	if err := ValidateID(name); err == nil {
		return fmt.Errorf("invalid repository name (%s), cannot specify 64-byte hexadecimal strings", name)
	}
	return nil
}

var validHex = regexp.MustCompile(`^([a-f0-9]{64})$`)

// ValidateID checks whether an ID string is a valid image ID.
func ValidateID(id string) error {
	if ok := validHex.MatchString(id); !ok {
		return fmt.Errorf("image ID '%s' is invalid ", id)
	}
	return nil
}
