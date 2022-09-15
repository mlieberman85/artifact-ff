//
// Copyright 2022 The GUAC Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package assembler

// ArtifactNode is a node that represents an artifact
type ArtifactNode struct {
	Name   string
	Digest string
}

func (an ArtifactNode) Type() string {
	return "Artifact"
}

func (an ArtifactNode) Properties() map[string]interface{} {
	properties := make(map[string]interface{})
	properties["name"] = an.Name
	properties["digest"] = an.Digest
	return properties
}

func (an ArtifactNode) PropertyNames() []string {
	return []string{"name", "digest"}
}

func (an ArtifactNode) IdentifiablePropertyNames() []string {
	// An artifact can be uniquely identified by digest
	return []string{"digest"}
}

// IdentityNode is a node that represents an identity
type IdentityNode struct {
	ID     string
	Digest string
	// base64 encoded
	Key       string
	KeyType   string
	KeyScheme string
}

func (in IdentityNode) Type() string {
	return "Identity"
}

func (in IdentityNode) Properties() map[string]interface{} {
	properties := make(map[string]interface{})
	properties["id"] = in.ID
	properties["digest"] = in.Digest
	properties["key"] = in.Key
	properties["keyType"] = in.KeyType
	properties["keyScheme"] = in.KeyScheme
	return properties
}

func (in IdentityNode) PropertyNames() []string {
	return []string{"id", "digest", "key", "keyType", "keyScheme"}
}

func (in IdentityNode) IdentifiablePropertyNames() []string {
	// An identity can be uniquely identified by digest
	return []string{"digest"}
}

// AttestationNode is a node that represents an attestation
type AttestationNode struct {
	// TODO(mihaimaruseac): Unsure what fields to store here
	FilePath string
	Digest   string
}

func (an AttestationNode) Type() string {
	return "Attestation"
}

func (an AttestationNode) Properties() map[string]interface{} {
	properties := make(map[string]interface{})
	properties["filepath"] = an.FilePath
	properties["digest"] = an.Digest
	return properties
}

func (an AttestationNode) PropertyNames() []string {
	return []string{"filepath", "digest"}
}

func (an AttestationNode) IdentifiablePropertyNames() []string {
	// An attestation can be uniquely identified by filename?
	return []string{"filepath"}
}

// BuilderNode is a node that represents a builder for an artifact
type BuilderNode struct {
	BuilderType string
	BuilderId   string
}

func (bn BuilderNode) Type() string {
	return "Builder"
}

func (bn BuilderNode) Properties() map[string]interface{} {
	properties := make(map[string]interface{})
	properties["type"] = bn.BuilderType
	properties["id"] = bn.BuilderId
	return properties
}

func (bn BuilderNode) PropertyNames() []string {
	return []string{"type", "id"}
}

func (bn BuilderNode) IdentifiablePropertyNames() []string {
	// A builder needs both type and id to be identified
	return []string{"type", "id"}
}

// IdentityForEdge is an edge that represents the fact that an
// `IdentityNode` is an identity for an `AttestationNode`.
type IdentityForEdge struct {
	IdentityNode    IdentityNode
	AttestationNode AttestationNode
}

func (e IdentityForEdge) Type() string {
	return "Identity"
}

func (e IdentityForEdge) Nodes() (v, u GuacNode) {
	return e.IdentityNode, e.AttestationNode
}

func (e IdentityForEdge) Properties() map[string]interface{} {
	return map[string]interface{}{}
}

func (e IdentityForEdge) PropertyNames() []string {
	return []string{}
}

func (e IdentityForEdge) IdentifiablePropertyNames() []string {
	return []string{}
}

// AttestationForEdge is an edge that represents the fact that an
// `AttestationNode` is an attestation for an `ArtifactNode`.
type AttestationForEdge struct {
	AttestationNode AttestationNode
	ArtifactNode    ArtifactNode
}

func (e AttestationForEdge) Type() string {
	return "Attestation"
}

func (e AttestationForEdge) Nodes() (v, u GuacNode) {
	return e.AttestationNode, e.ArtifactNode
}

func (e AttestationForEdge) Properties() map[string]interface{} {
	return map[string]interface{}{}
}

func (e AttestationForEdge) PropertyNames() []string {
	return []string{}
}

func (e AttestationForEdge) IdentifiablePropertyNames() []string {
	return []string{}
}

// BuiltByEdge is an edge that represents the fact that an
// `ArtifactNode` has been built by a `BuilderNode`
type BuiltByEdge struct {
	ArtifactNode ArtifactNode
	BuilderNode  BuilderNode
}

func (e BuiltByEdge) Type() string {
	return "BuiltBy"
}

func (e BuiltByEdge) Nodes() (v, u GuacNode) {
	return e.ArtifactNode, e.BuilderNode
}

func (e BuiltByEdge) Properties() map[string]interface{} {
	return map[string]interface{}{}
}

func (e BuiltByEdge) PropertyNames() []string {
	return []string{}
}

func (e BuiltByEdge) IdentifiablePropertyNames() []string {
	return []string{}
}

// DependsOnEdge is an edge that represents the fact that an
// `ArtifactNode` depends on another `ArtifactNode`
type DependsOnEdge struct {
	ArtifactNode ArtifactNode
	Dependency   ArtifactNode
}

func (e DependsOnEdge) Type() string {
	return "DependsOn"
}

func (e DependsOnEdge) Nodes() (v, u GuacNode) {
	return e.ArtifactNode, e.Dependency
}

func (e DependsOnEdge) Properties() map[string]interface{} {
	return map[string]interface{}{}
}

func (e DependsOnEdge) PropertyNames() []string {
	return []string{}
}

func (e DependsOnEdge) IdentifiablePropertyNames() []string {
	return []string{}
}