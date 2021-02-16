package trust

import (
	"encoding/hex"
	"path"
	"sort"
	"strings"

	"github.com/fvbommel/sortorder"
	"github.com/theupdateframework/notary"
	"github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/tuf/data"
)

var (
	// ReleasesRole is the role named "releases"
	ReleasesRole = data.RoleName(path.Join(data.CanonicalTargetsRole.String(), "releases"))
)

// isReleasedTarget checks if a role name is "released":
// either targets/releases or targets TUF roles
func isReleasedTarget(role data.RoleName) bool {
	return role == data.CanonicalTargetsRole || role == ReleasesRole
}

const releasedRoleName = "Repo Admin"

// notaryRoleToSigner converts TUF role name to a human-understandable signer name
func notaryRoleToSigner(tufRole data.RoleName) string {
	//  don't show a signer for "targets" or "targets/releases"
	if isReleasedTarget(data.RoleName(tufRole.String())) {
		return releasedRoleName
	}
	return strings.TrimPrefix(tufRole.String(), "targets/")
}

// aggregate all signers for a "released" hash+tagname pair. To be "released," the tag must have been
// signed into the "targets" or "targets/releases" role. Output is sorted by tag name
func matchReleasedSignatures(allTargets []client.TargetSignedStruct) []trustTagRow {
	signatureRows := []trustTagRow{}
	// do a first pass to get filter on tags signed into "targets" or "targets/releases"
	releasedTargetRows := map[trustTagKey][]string{}
	for _, tgt := range allTargets {
		if isReleasedTarget(tgt.Role.Name) {
			releasedKey := trustTagKey{tgt.Target.Name, hex.EncodeToString(tgt.Target.Hashes[notary.SHA256])}
			releasedTargetRows[releasedKey] = []string{}
		}
	}

	// now fill out all signers on released keys
	for _, tgt := range allTargets {
		targetKey := trustTagKey{tgt.Target.Name, hex.EncodeToString(tgt.Target.Hashes[notary.SHA256])}
		// only considered released targets
		if _, ok := releasedTargetRows[targetKey]; ok && !isReleasedTarget(tgt.Role.Name) {
			releasedTargetRows[targetKey] = append(releasedTargetRows[targetKey], notaryRoleToSigner(tgt.Role.Name))
		}
	}

	// compile the final output as a sorted slice
	for targetKey, signers := range releasedTargetRows {
		signatureRows = append(signatureRows, trustTagRow{targetKey, signers})
	}
	sort.Slice(signatureRows, func(i, j int) bool {
		return sortorder.NaturalLess(signatureRows[i].SignedTag, signatureRows[j].SignedTag)
	})
	return signatureRows
}

func getDelegationRoleToKeyMap(rawDelegationRoles []data.Role) map[string][]string {
	signerRoleToKeyIDs := make(map[string][]string)
	for _, delRole := range rawDelegationRoles {
		switch delRole.Name {
		case ReleasesRole, data.CanonicalRootRole, data.CanonicalSnapshotRole, data.CanonicalTargetsRole, data.CanonicalTimestampRole:
			continue
		default:
			signerRoleToKeyIDs[notaryRoleToSigner(delRole.Name)] = delRole.KeyIDs
		}
	}
	return signerRoleToKeyIDs
}
