package v2

// Project
// Metadata:
//		public: "true" / "false"
type Project struct {
	ID       int64                  `json:"project_id"`
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata"`
}

type Repository struct {
	Name string `json:"name"`
}

type Artifact struct {
	ID   int64  `json:"id"`
	Tags []Tag  `json:"tags"`
	Type string `json:"type,omitempty"`
}

type Tag struct {
	ID         int64  `json:"id"`
	ArtifactID int64  `json:"artifact_id"`
	Name       string `json:"name"`
}
