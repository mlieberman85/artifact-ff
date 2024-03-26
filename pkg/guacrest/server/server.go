//
// Copyright 2024 The GUAC Authors.
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

package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/Khan/genqlient/graphql"
	"github.com/guacsec/guac/internal/testing/ptrfrom"
	model "github.com/guacsec/guac/pkg/assembler/clients/generated"
	"github.com/guacsec/guac/pkg/assembler/helpers"
	gen "github.com/guacsec/guac/pkg/guacrest/generated"
	"github.com/guacsec/guac/pkg/misc/depversion"
	"go.uber.org/zap"
)

const (
	hashEqualStr        string = "hashEqual"
	scorecardStr        string = "scorecard"
	occurrenceStr       string = "occurrence"
	hasSrcAtStr         string = "hasSrcAt"
	hasSBOMStr          string = "hasSBOM"
	hasSLSAStr          string = "hasSLSA"
	certifyVulnStr      string = "certifyVuln"
	vexLinkStr          string = "vexLink"
	badLinkStr          string = "badLink"
	goodLinkStr         string = "goodLink"
	pkgEqualStr         string = "pkgEqual"
	packageSubjectType  string = "package"
	sourceSubjectType   string = "source"
	artifactSubjectType string = "artifact"
	guacType            string = "guac"
	noVulnType          string = "novuln"
)

// DefaultServer implements the API, backed by the GraphQL Server
type DefaultServer struct {
	gqlClient graphql.Client
	logger    *zap.SugaredLogger
}

func NewDefaultServer(gqlClient graphql.Client, logger *zap.SugaredLogger) *DefaultServer {
	return &DefaultServer{gqlClient: gqlClient, logger: logger}
}

func (s *DefaultServer) HealthCheck(ctx context.Context, request gen.HealthCheckRequestObject) (gen.HealthCheckResponseObject, error) {
	return gen.HealthCheck200JSONResponse("Server is healthy"), nil
}

func (s *DefaultServer) AnalyzeDependencies(ctx context.Context, request gen.AnalyzeDependenciesRequestObject) (gen.AnalyzeDependenciesResponseObject, error) {
	return nil, fmt.Errorf("Unimplemented")
}

func (s *DefaultServer) RetrieveDependencies(ctx context.Context, request gen.RetrieveDependenciesRequestObject) (gen.RetrieveDependenciesResponseObject, error) {
	return nil, fmt.Errorf("Unimplemented")
}

func (s *DefaultServer) GetArtifactInformation(ctx context.Context, request gen.GetArtifactInformationRequestObject) (gen.GetArtifactInformationResponseObject, error) {
	logger := s.logger
	artResponse, err := getArtifactResponse(ctx, s.gqlClient, request.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hash: %w", err)
	}
	// if artifact is not found, return nothing
	if artResponse == nil {
		return nil, nil
	}
	logger.Infof("artifact found with ID %s", artResponse)

	neighborResponse, err := model.Neighbors(ctx, s.gqlClient, artResponse.Artifacts[0].Id, []model.Edge{})
	if err != nil {
		return nil, fmt.Errorf("error querying neighbors: %v", err)
	}

	var foundSbomList []gen.Sbom
	var foundSlsaList []gen.Slsa
	var foundVulnerabilities []gen.Vulnerability
	var foundBads []string

	for _, neighbor := range neighborResponse.Neighbors {
		switch v := neighbor.(type) {
		case *model.NeighborsNeighborsHasSBOM:
			foundSbomList = append(foundSbomList, v.DownloadLocation)
		case *model.NeighborsNeighborsHasSLSA:
			foundSlsaList = append(foundSlsaList, v.Slsa.Origin)
		case *model.NeighborsNeighborsIsOccurrence:
			// if there is an isOccurrence, check to see if there are slsa attestation associated with it
			switch sub := v.Subject.(type) {
			case *model.AllIsOccurrencesTreeSubjectPackage:
				logger.Infof("querying for vulns via pkg ID: %s", sub.Namespaces[0].Names[0].Versions[0].Id)

				vulnsList, badlist, err := searchPkgViaHasSBOM(ctx, s.gqlClient, sub.Namespaces[0].Names[0].Versions[0].Id, 0, true)
				if err != nil {
					return nil, fmt.Errorf("query vulns via hasSBOM failed: %v", err)
				}

				foundBads = append(foundBads, badlist...)
				foundVulnerabilities = append(foundVulnerabilities, vulnsList...)

				neighborResponseSBOM, err := model.Neighbors(ctx, s.gqlClient, sub.Namespaces[0].Names[0].Versions[0].Id, []model.Edge{model.EdgePackageHasSbom})
				if err != nil {
					continue
				} else {
					for _, neighborHasSBOM := range neighborResponseSBOM.Neighbors {
						if hasSBOM, ok := neighborHasSBOM.(*model.NeighborsNeighborsHasSBOM); ok {
							foundSbomList = append(foundSbomList, hasSBOM.DownloadLocation)
						}
					}
				}

			case *model.AllIsOccurrencesTreeSubjectSource:
				continue
			}

		default:
			continue
		}
	}
	logger.Infof("returning results")
	val := gen.GetArtifactInformation200JSONResponse{
		InfoJSONResponse: gen.InfoJSONResponse{
			SbomList:        foundSbomList,
			SlsaList:        foundSlsaList,
			Vulnerabilities: foundVulnerabilities,
			CertifyBads:     foundBads,
		},
	}

	return val, nil
}

func (s *DefaultServer) GetArtifactSbomInformation(ctx context.Context, request gen.GetArtifactSbomInformationRequestObject) (gen.GetArtifactSbomInformationResponseObject, error) {
	artResponse, err := getArtifactResponse(ctx, s.gqlClient, request.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hash: %w", err)
	}

	// if artifact is not found, return nothing
	if artResponse == nil {
		return nil, nil
	}

	neighborResponse, err := model.Neighbors(ctx, s.gqlClient, artResponse.Artifacts[0].Id, []model.Edge{model.EdgeArtifactHasSbom, model.EdgeArtifactIsOccurrence})
	if err != nil {
		return nil, fmt.Errorf("error querying neighbors: %v", err)
	}

	var foundSbomList []gen.Sbom

	for _, neighbor := range neighborResponse.Neighbors {
		switch v := neighbor.(type) {
		case *model.NeighborsNeighborsHasSBOM:
			foundSbomList = append(foundSbomList, v.DownloadLocation)
		case *model.NeighborsNeighborsIsOccurrence:
			// if there is an isOccurrence, check to see if there are slsa attestation associated with it
			switch sub := v.Subject.(type) {
			case *model.AllIsOccurrencesTreeSubjectPackage:
				neighborResponseSBOM, err := model.Neighbors(ctx, s.gqlClient, sub.Namespaces[0].Names[0].Versions[0].Id, []model.Edge{model.EdgePackageHasSbom})
				if err != nil {
					continue
				} else {
					for _, neighborHasSBOM := range neighborResponseSBOM.Neighbors {
						if hasSBOM, ok := neighborHasSBOM.(*model.NeighborsNeighborsHasSBOM); ok {
							foundSbomList = append(foundSbomList, hasSBOM.DownloadLocation)
						}
					}
				}
			case *model.AllIsOccurrencesTreeSubjectSource:
				continue
			}

		default:
			continue
		}
	}

	val := gen.GetArtifactSbomInformation200JSONResponse{
		SbomInfoJSONResponse: gen.SbomInfoJSONResponse{
			SbomList: foundSbomList,
		},
	}

	return val, nil

}
func (s *DefaultServer) GetArtifactSlsaInformation(ctx context.Context, request gen.GetArtifactSlsaInformationRequestObject) (gen.GetArtifactSlsaInformationResponseObject, error) {
	artResponse, err := getArtifactResponse(ctx, s.gqlClient, request.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hash: %w", err)
	}

	// if artifact is not found, return nothing
	if artResponse == nil {
		return nil, nil
	}

	neighborResponse, err := model.Neighbors(ctx, s.gqlClient, artResponse.Artifacts[0].Id, []model.Edge{model.EdgeArtifactHasSlsa})
	if err != nil {
		return nil, fmt.Errorf("error querying neighbors: %v", err)
	}

	var foundSlsaList []gen.Slsa

	for _, neighbor := range neighborResponse.Neighbors {
		switch v := neighbor.(type) {
		case *model.NeighborsNeighborsHasSLSA:
			foundSlsaList = append(foundSlsaList, v.Slsa.Origin)
		default:
			continue
		}
	}

	val := gen.GetArtifactSlsaInformation200JSONResponse{
		SlsaInfoJSONResponse: gen.SlsaInfoJSONResponse{
			SlsaList: foundSlsaList,
		},
	}
	return val, nil
}

func (s *DefaultServer) GetArtifactVulnInformation(ctx context.Context, request gen.GetArtifactVulnInformationRequestObject) (gen.GetArtifactVulnInformationResponseObject, error) {
	artResponse, err := getArtifactResponse(ctx, s.gqlClient, request.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to parse hash: %w", err)
	}

	// if artifact is not found, return nothing
	if artResponse == nil {
		return nil, nil
	}

	neighborResponse, err := model.Neighbors(ctx, s.gqlClient, artResponse.Artifacts[0].Id, []model.Edge{model.EdgeArtifactIsOccurrence})
	if err != nil {
		return nil, fmt.Errorf("error querying neighbors: %v", err)
	}

	var foundVulnerabilities []gen.Vulnerability
	var foundBads []string
	for _, neighbor := range neighborResponse.Neighbors {
		switch v := neighbor.(type) {
		case *model.NeighborsNeighborsIsOccurrence:
			// if there is an isOccurrence, check to see if there are slsa attestation associated with it
			switch sub := v.Subject.(type) {
			case *model.AllIsOccurrencesTreeSubjectPackage:
				vulnsList, badList, err := searchPkgViaHasSBOM(ctx, s.gqlClient, sub.Namespaces[0].Names[0].Versions[0].Id, 0, true)
				if err != nil {
					return nil, fmt.Errorf("query vulns via hasSBOM failed: %v", err)
				}

				foundVulnerabilities = append(foundVulnerabilities, vulnsList...)
				foundBads = append(foundBads, badList...)

			case *model.AllIsOccurrencesTreeSubjectSource:
				continue
			}

		default:
			continue
		}
	}

	val := gen.GetArtifactVulnInformation200JSONResponse{
		VulnInfoJSONResponse: gen.VulnInfoJSONResponse{
			Vulnerabilities: foundVulnerabilities,
			CertifyBads:     foundBads,
		},
	}

	return val, nil
}

func (s *DefaultServer) GetPackageSbomInformation(ctx context.Context, request gen.GetPackageSbomInformationRequestObject) (gen.GetPackageSbomInformationResponseObject, error) {
	pkgResponse, err := getPkgResponseFromPurl(ctx, s.gqlClient, request.Purl)
	if err != nil {
		return nil, fmt.Errorf("error querying for package: %v", err)
	}
	if len(pkgResponse.Packages) != 1 {
		return nil, fmt.Errorf("failed to located package based on purl")
	}
	neighborResponse, err := model.Neighbors(ctx, s.gqlClient, pkgResponse.Packages[0].Namespaces[0].Names[0].Versions[0].Id, []model.Edge{model.EdgePackageHasSbom})
	if err != nil {
		return nil, fmt.Errorf("error querying neighbors: %v", err)
	}

	var foundSbomList []gen.Sbom

	for _, neighbor := range neighborResponse.Neighbors {
		switch v := neighbor.(type) {
		case *model.NeighborsNeighborsHasSBOM:
			foundSbomList = append(foundSbomList, v.DownloadLocation)
		default:
			continue
		}
	}

	val := gen.GetPackageSbomInformation200JSONResponse{
		SbomInfoJSONResponse: gen.SbomInfoJSONResponse{
			SbomList: foundSbomList,
		},
	}

	return val, nil
}

func (s *DefaultServer) GetPackageSlsaInformation(ctx context.Context, request gen.GetPackageSlsaInformationRequestObject) (gen.GetPackageSlsaInformationResponseObject, error) {
	pkgResponse, err := getPkgResponseFromPurl(ctx, s.gqlClient, request.Purl)
	if err != nil {
		return nil, fmt.Errorf("error querying for package: %v", err)
	}
	if len(pkgResponse.Packages) != 1 {
		return nil, fmt.Errorf("failed to located package based on purl")
	}
	neighborResponse, err := model.Neighbors(ctx, s.gqlClient, pkgResponse.Packages[0].Namespaces[0].Names[0].Versions[0].Id, []model.Edge{model.EdgePackageIsOccurrence})
	if err != nil {
		return nil, fmt.Errorf("error querying neighbors: %v", err)
	}

	var foundSlsaList []gen.Slsa

	for _, neighbor := range neighborResponse.Neighbors {
		switch v := neighbor.(type) {
		case *model.NeighborsNeighborsIsOccurrence:
			// if there is an isOccurrence, check to see if there are slsa attestation associated with it
			artifactFilter := &model.ArtifactSpec{
				Algorithm: &v.Artifact.Algorithm,
				Digest:    &v.Artifact.Digest,
			}
			artifactResponse, err := model.Artifacts(ctx, s.gqlClient, *artifactFilter)
			if err != nil {
				continue
			}
			if len(artifactResponse.Artifacts) != 1 {
				continue
			}
			neighborResponseHasSLSA, err := model.Neighbors(ctx, s.gqlClient, artifactResponse.Artifacts[0].Id, []model.Edge{model.EdgeArtifactHasSlsa})
			if err != nil {
				continue
			} else {
				for _, neighborHasSLSA := range neighborResponseHasSLSA.Neighbors {
					if hasSLSA, ok := neighborHasSLSA.(*model.NeighborsNeighborsHasSLSA); ok {
						foundSlsaList = append(foundSlsaList, hasSLSA.Slsa.Origin)
					}
				}
			}
		default:
			continue
		}
	}

	val := gen.GetPackageSlsaInformation200JSONResponse{
		SlsaInfoJSONResponse: gen.SlsaInfoJSONResponse{
			SlsaList: foundSlsaList,
		},
	}

	return val, nil

}

func (s *DefaultServer) GetPackageVulnInformation(ctx context.Context, request gen.GetPackageVulnInformationRequestObject) (gen.GetPackageVulnInformationResponseObject, error) {

	pkgResponse, err := getPkgResponseFromPurl(ctx, s.gqlClient, request.Purl)
	if err != nil {
		return nil, fmt.Errorf("error querying for package: %v", err)
	}
	if len(pkgResponse.Packages) != 1 {
		return nil, fmt.Errorf("failed to located package based on purl")
	}
	neighborResponse, err := model.Neighbors(ctx, s.gqlClient, pkgResponse.Packages[0].Namespaces[0].Names[0].Versions[0].Id, []model.Edge{model.EdgePackageCertifyVuln})
	if err != nil {
		return nil, fmt.Errorf("error querying neighbors: %v", err)
	}

	var foundVulnerabilities []gen.Vulnerability

	for _, neighbor := range neighborResponse.Neighbors {
		switch v := neighbor.(type) {
		case *model.NeighborsNeighborsCertifyVuln:
			foundVulnerabilities = append(foundVulnerabilities, v.Vulnerability.VulnerabilityIDs[0].VulnerabilityID)
		default:
			continue
		}
	}

	vulnsList, badList, err := searchPkgViaHasSBOM(ctx, s.gqlClient, pkgResponse.Packages[0].Namespaces[0].Names[0].Versions[0].Id, 0, true)
	if err != nil {
		return nil, fmt.Errorf("query vulns via hasSBOM failed: %v", err)
	}

	foundVulnerabilities = append(foundVulnerabilities, vulnsList...)

	val := gen.GetPackageVulnInformation200JSONResponse{
		VulnInfoJSONResponse: gen.VulnInfoJSONResponse{
			Vulnerabilities: foundVulnerabilities,
			CertifyBads:     badList,
		},
	}

	return val, nil
}

func (s *DefaultServer) GetPackageInformation(ctx context.Context, request gen.GetPackageInformationRequestObject) (gen.GetPackageInformationResponseObject, error) {

	pkgResponse, err := getPkgResponseFromPurl(ctx, s.gqlClient, request.Purl)
	if err != nil {
		return nil, fmt.Errorf("error querying for package: %v", err)
	}
	if len(pkgResponse.Packages) != 1 {
		return nil, fmt.Errorf("failed to located package based on purl")
	}
	neighborResponse, err := model.Neighbors(ctx, s.gqlClient, pkgResponse.Packages[0].Namespaces[0].Names[0].Versions[0].Id, []model.Edge{})
	if err != nil {
		return nil, fmt.Errorf("error querying neighbors: %v", err)
	}

	var foundSbomList []gen.Sbom
	var foundSlsaList []gen.Slsa
	var foundVulnerabilities []gen.Vulnerability

	for _, neighbor := range neighborResponse.Neighbors {
		switch v := neighbor.(type) {
		case *model.NeighborsNeighborsCertifyVuln:
			foundVulnerabilities = append(foundVulnerabilities, v.Vulnerability.VulnerabilityIDs[0].VulnerabilityID)
		case *model.NeighborsNeighborsHasSBOM:
			foundSbomList = append(foundSbomList, v.DownloadLocation)
		case *model.NeighborsNeighborsIsOccurrence:
			// if there is an isOccurrence, check to see if there are slsa attestation associated with it
			neighborResponseHasSLSA, err := model.Neighbors(ctx, s.gqlClient, v.Artifact.Id, []model.Edge{model.EdgeArtifactHasSlsa})
			if err != nil {
				continue
			} else {
				for _, neighborHasSLSA := range neighborResponseHasSLSA.Neighbors {
					if hasSLSA, ok := neighborHasSLSA.(*model.NeighborsNeighborsHasSLSA); ok {
						foundSlsaList = append(foundSlsaList, hasSLSA.Slsa.Origin)
					}
				}
			}
		default:
			continue
		}
	}

	vulnsList, badsList, err := searchPkgViaHasSBOM(ctx, s.gqlClient, pkgResponse.Packages[0].Namespaces[0].Names[0].Versions[0].Id, 0, true)
	if err != nil {
		return nil, fmt.Errorf("query vulns via hasSBOM failed: %v", err)
	}

	foundVulnerabilities = append(foundVulnerabilities, vulnsList...)

	val := gen.GetPackageInformation200JSONResponse{
		InfoJSONResponse: gen.InfoJSONResponse{
			SbomList:        foundSbomList,
			SlsaList:        foundSlsaList,
			Vulnerabilities: foundVulnerabilities,
			CertifyBads:     badsList,
		},
	}

	return val, nil
}

func getArtifactResponse(ctx context.Context, gqlclient graphql.Client, subject string) (*model.ArtifactsResponse, error) {
	split := strings.Split(subject, ":")
	if len(split) != 2 {
		return nil, fmt.Errorf("failed to parse artifact. Needs to be in algorithm:digest form")
	}
	artifactFilter := &model.ArtifactSpec{
		Algorithm: ptrfrom.String(strings.ToLower(string(split[0]))),
		Digest:    ptrfrom.String(strings.ToLower(string(split[1]))),
	}

	artifactResponse, err := model.Artifacts(ctx, gqlclient, *artifactFilter)
	if err != nil {
		return nil, err
	}
	if len(artifactResponse.Artifacts) != 1 {
		return nil, err
	}
	return artifactResponse, nil
}

// func concurrentVulnAndVexNeighbors(ctx context.Context, gqlclient graphql.Client, pkgID string, isDep model.AllHasSBOMTreeIncludedDependenciesIsDependency, resultChan chan<- struct {
// 	pkgVersionNeighborResponse *model.NeighborsResponse
// 	isDep                      model.AllHasSBOMTreeIncludedDependenciesIsDependency
// }, wg *sync.WaitGroup, errChan chan<- error) {

// 	defer wg.Done()

// 	pkgVersionNeighborResponse, err := model.Neighbors(ctx, gqlclient, pkgID, []model.Edge{model.EdgePackageCertifyVuln, model.EdgePackageCertifyVexStatement})
// 	if err != nil {
// 		errChan <- fmt.Errorf("error querying neighbor via pkgID: %s for vulnerability: %w", pkgID, err)
// 		return
// 	}

// 	// Send the results to the resultChan
// 	resultChan <- struct {
// 		pkgVersionNeighborResponse *model.NeighborsResponse
// 		isDep                      model.AllHasSBOMTreeIncludedDependenciesIsDependency
// 	}{pkgVersionNeighborResponse, isDep}
// }

// searchPkgViaHasSBOM takes in either a purl or URI for the initial value to find the hasSBOM node.
// From there is recursively searches through all the dependencies to determine if it contains hasSBOM nodes.
// It concurrent checks the package version node if it contains vulnerabilities and VEX data.
func searchPkgViaHasSBOM(ctx context.Context, gqlclient graphql.Client, pkgVersionID string, maxLength int, isPurl bool) ([]string, []string, error) {
	checkedPkgIDs := make(map[string]bool)
	var foundVulns []string

	//var wg sync.WaitGroup

	queue := make([]string, 0) // the queue of nodes in bfs
	type dfsNode struct {
		expanded bool // true once all node neighbors are added to queue
		parent   string
		pkgID    string
		depth    int
	}
	nodeMap := map[string]dfsNode{}

	nodeMap[pkgVersionID] = dfsNode{}
	queue = append(queue, pkgVersionID)

	// resultChan := make(chan struct {
	// 	pkgVersionNeighborResponse *model.NeighborsResponse
	// 	isDep                      model.AllHasSBOMTreeIncludedDependenciesIsDependency
	// })

	// errChan := make(chan error)

	checkedCertifyVulnIDs := make(map[string]bool)

	neighborVulns := make(map[string]*model.NeighborsNeighborsCertifyVuln)
	neighborVex := make(map[string]*model.NeighborsNeighborsCertifyVEXStatement)
	var foundCertifyBad []string

	for len(queue) > 0 {
		now := queue[0]
		queue = queue[1:]
		nowNode := nodeMap[now]

		if maxLength != 0 && nowNode.depth >= maxLength {
			break
		}

		var foundHasSBOMPkg *model.HasSBOMsResponse
		var err error

		// if the initial depth, check if its a purl or an SBOM URI. Otherwise always search by pkgID
		if nowNode.depth == 0 {
			if isPurl {
				foundHasSBOMPkg, err = model.HasSBOMs(ctx, gqlclient, model.HasSBOMSpec{Subject: &model.PackageOrArtifactSpec{Package: &model.PkgSpec{Id: &pkgVersionID}}})
				if err != nil {
					return nil, nil, fmt.Errorf("failed getting hasSBOM via purl: %s with error :%w", now, err)
				}
			} else {
				foundHasSBOMPkg, err = model.HasSBOMs(ctx, gqlclient, model.HasSBOMSpec{Uri: &now})
				if err != nil {
					return nil, nil, fmt.Errorf("failed getting hasSBOM via URI: %s with error: %w", now, err)
				}
			}
		} else {
			foundHasSBOMPkg, err = model.HasSBOMs(ctx, gqlclient, model.HasSBOMSpec{Subject: &model.PackageOrArtifactSpec{Package: &model.PkgSpec{Id: &now}}})
			if err != nil {
				return nil, nil, fmt.Errorf("failed getting hasSBOM via ID: %s with error :%w", now, err)
			}
		}

		for _, hasSBOM := range foundHasSBOMPkg.HasSBOM {
			if pkgResponse, ok := foundHasSBOMPkg.HasSBOM[0].Subject.(*model.AllHasSBOMTreeSubjectPackage); ok {
				if pkgResponse.Type != guacType {
					if !checkedPkgIDs[pkgResponse.Namespaces[0].Names[0].Versions[0].Id] {
						vulnPath, bads, err := queryVulnsViaPackageNeighbors(ctx, gqlclient, pkgResponse.Namespaces[0].Names[0].Versions[0].Id)
						if err != nil {
							return nil, nil, fmt.Errorf("error querying neighbor: %v", err)
						}

						foundVulns = append(foundVulns, vulnPath...)
						foundCertifyBad = append(foundCertifyBad, bads...)
						checkedPkgIDs[pkgResponse.Namespaces[0].Names[0].Versions[0].Id] = true
					}
				}
			}
			for _, isDep := range hasSBOM.IncludedDependencies {
				if isDep.DependencyPackage.Type == guacType {
					continue
				}
				var matchingDepPkgVersionIDs []string
				if len(isDep.DependencyPackage.Namespaces[0].Names[0].Versions) == 0 {
					findMatchingDepPkgVersionIDs, err := findDepPkgVersionIDs(ctx, gqlclient, isDep.DependencyPackage.Type, isDep.DependencyPackage.Namespaces[0].Namespace,
						isDep.DependencyPackage.Namespaces[0].Names[0].Name, isDep.VersionRange)
					if err != nil {
						return nil, nil, fmt.Errorf("error from findMatchingDepPkgVersionIDs:%w", err)
					}
					matchingDepPkgVersionIDs = append(matchingDepPkgVersionIDs, findMatchingDepPkgVersionIDs...)
				} else {
					matchingDepPkgVersionIDs = append(matchingDepPkgVersionIDs, isDep.DependencyPackage.Namespaces[0].Names[0].Versions[0].Id)
				}
				for _, pkgID := range matchingDepPkgVersionIDs {
					dfsN, seen := nodeMap[pkgID]
					if !seen {
						dfsN = dfsNode{
							parent: now,
							pkgID:  pkgID,
							depth:  nowNode.depth + 1,
						}
						nodeMap[pkgID] = dfsN
					}
					if !dfsN.expanded {
						queue = append(queue, pkgID)
					}
					pkgVersionNeighborResponse, err := model.Neighbors(ctx, gqlclient, pkgID, []model.Edge{model.EdgePackageCertifyVuln, model.EdgePackageCertifyVexStatement, model.EdgePackageCertifyBad})
					if err != nil {
						return nil, nil, fmt.Errorf("error getting package neighbors:%w", err)
					}

					for _, neighbor := range pkgVersionNeighborResponse.Neighbors {
						if certifyVuln, ok := neighbor.(*model.NeighborsNeighborsCertifyVuln); ok {
							if !checkedCertifyVulnIDs[certifyVuln.Id] {
								if certifyVuln.Vulnerability.Type != noVulnType {
									checkedCertifyVulnIDs[certifyVuln.Id] = true
									for _, vuln := range certifyVuln.Vulnerability.VulnerabilityIDs {
										neighborVulns[vuln.Id] = certifyVuln
									}
								}
							}
						}

						if certifyVex, ok := neighbor.(*model.NeighborsNeighborsCertifyVEXStatement); ok {
							for _, vuln := range certifyVex.Vulnerability.VulnerabilityIDs {
								neighborVex[vuln.Id] = certifyVex
							}
						}

						if certifyBad, ok := neighbor.(*model.NeighborsNeighborsCertifyBad); ok {
							foundCertifyBad = append(foundCertifyBad, certifyBad.Justification)
						}
					}

					checkedPkgIDs[pkgID] = true
				}
			}
		}
		nowNode.expanded = true
		nodeMap[now] = nowNode
	}

	// // Close the result channel once all goroutines are done
	// go func() {
	// 	wg.Wait()
	// 	close(resultChan)
	// 	close(errChan)
	// }()

	// Collect results from the channel

	// for result := range resultChan {
	// 	for _, neighbor := range result.pkgVersionNeighborResponse.Neighbors {
	// 		if certifyVuln, ok := neighbor.(*model.NeighborsNeighborsCertifyVuln); ok {
	// 			if !checkedCertifyVulnIDs[certifyVuln.Id] {
	// 				if certifyVuln.Vulnerability.Type != noVulnType {
	// 					checkedCertifyVulnIDs[certifyVuln.Id] = true
	// 					for _, vuln := range certifyVuln.Vulnerability.VulnerabilityIDs {
	// 						neighborVulns[vuln.Id] = certifyVuln
	// 					}
	// 				}
	// 			}
	// 		}

	// 		if certifyVex, ok := neighbor.(*model.NeighborsNeighborsCertifyVEXStatement); ok {
	// 			for _, vuln := range certifyVex.Vulnerability.VulnerabilityIDs {
	// 				neighborVex[vuln.Id] = certifyVex
	// 			}
	// 		}
	// 	}
	// }

	// for err := range errChan {
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	foundVulns = append(foundVulns, removeVulnsWithValidVex(neighborVex, neighborVulns)...)

	return foundVulns, foundCertifyBad, nil
}

func findDepPkgVersionIDs(ctx context.Context, gqlclient graphql.Client, depPkgType string, depPkgNameSpace string, depPkgName string, versionRange string) ([]string, error) {
	var matchingDepPkgVersionIDs []string

	depPkgFilter := &model.PkgSpec{
		Type:      &depPkgType,
		Namespace: &depPkgNameSpace,
		Name:      &depPkgName,
	}

	depPkgResponse, err := model.Packages(ctx, gqlclient, *depPkgFilter)
	if err != nil {
		return nil, fmt.Errorf("error querying for dependent package: %w", err)
	}

	depPkgVersionsMap := map[string]string{}
	depPkgVersions := []string{}
	for _, depPkgVersion := range depPkgResponse.Packages[0].Namespaces[0].Names[0].Versions {
		depPkgVersions = append(depPkgVersions, depPkgVersion.Version)
		depPkgVersionsMap[depPkgVersion.Version] = depPkgVersion.Id
	}

	matchingDepPkgVersions, err := depversion.WhichVersionMatches(depPkgVersions, versionRange)
	if err != nil {
		// TODO(jeffmendoza): depversion is not handling all/new possible
		// version ranges from deps.dev. Continue here to report possible
		// vulns even if some paths cannot be followed.
		matchingDepPkgVersions = nil
		//return nil, nil, fmt.Errorf("error determining dependent version matches: %w", err)
	}

	for matchingDepPkgVersion := range matchingDepPkgVersions {
		matchingDepPkgVersionIDs = append(matchingDepPkgVersionIDs, depPkgVersionsMap[matchingDepPkgVersion])
	}
	return matchingDepPkgVersionIDs, nil
}

func getPkgResponseFromPurl(ctx context.Context, gqlclient graphql.Client, purl string) (*model.PackagesResponse, error) {
	pkgInput, err := helpers.PurlToPkg(purl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PURL: %v", err)
	}

	pkgQualifierFilter := []model.PackageQualifierSpec{}
	for _, qualifier := range pkgInput.Qualifiers {
		// to prevent https://github.com/golang/go/discussions/56010
		qualifier := qualifier
		pkgQualifierFilter = append(pkgQualifierFilter, model.PackageQualifierSpec{
			Key:   qualifier.Key,
			Value: &qualifier.Value,
		})
	}

	pkgFilter := &model.PkgSpec{
		Type:       &pkgInput.Type,
		Namespace:  pkgInput.Namespace,
		Name:       &pkgInput.Name,
		Version:    pkgInput.Version,
		Subpath:    pkgInput.Subpath,
		Qualifiers: pkgQualifierFilter,
	}
	pkgResponse, err := model.Packages(ctx, gqlclient, *pkgFilter)
	if err != nil {
		return nil, fmt.Errorf("error querying for package: %v", err)
	}
	if len(pkgResponse.Packages) != 1 {
		return nil, fmt.Errorf("failed to located package based on purl")
	}
	return pkgResponse, nil
}

func queryVulnsViaPackageNeighbors(ctx context.Context, gqlclient graphql.Client, pkgVersionID string) ([]string, []string, error) {
	neighborVulns := make(map[string]*model.NeighborsNeighborsCertifyVuln)
	neighborVex := make(map[string]*model.NeighborsNeighborsCertifyVEXStatement)

	var foundCertifyBad []string

	var edgeTypes = []model.Edge{model.EdgePackageCertifyVuln, model.EdgePackageCertifyVexStatement, model.EdgePackageCertifyBad}

	pkgVersionNeighborResponse, err := model.Neighbors(ctx, gqlclient, pkgVersionID, edgeTypes)
	if err != nil {
		return nil, nil, fmt.Errorf("error querying neighbor for vulnerability: %w", err)
	}
	certifyVulnFound := false
	for _, neighbor := range pkgVersionNeighborResponse.Neighbors {
		if certifyVuln, ok := neighbor.(*model.NeighborsNeighborsCertifyVuln); ok {
			certifyVulnFound = true
			if certifyVuln.Vulnerability.Type != noVulnType {
				for _, vuln := range certifyVuln.Vulnerability.VulnerabilityIDs {
					neighborVulns[vuln.Id] = certifyVuln
				}
			}
		}

		if certifyVex, ok := neighbor.(*model.NeighborsNeighborsCertifyVEXStatement); ok {
			for _, vuln := range certifyVex.Vulnerability.VulnerabilityIDs {
				neighborVex[vuln.Id] = certifyVex
			}
		}

		if certifyBad, ok := neighbor.(*model.NeighborsNeighborsCertifyBad); ok {
			foundCertifyBad = append(foundCertifyBad, certifyBad.Justification)
		}
	}
	if !certifyVulnFound {
		return nil, nil, fmt.Errorf("error certify vulnerability node not found, incomplete data. Please ensure certifier has run by running guacone certifier osv")
	}

	return removeVulnsWithValidVex(neighborVex, neighborVulns), foundCertifyBad, nil
}

func removeVulnsWithValidVex(neighborVex map[string]*model.NeighborsNeighborsCertifyVEXStatement, neighborVulns map[string]*model.NeighborsNeighborsCertifyVuln) []string {
	var vulns []string

	for _, foundVuln := range neighborVulns {
		foundVuln := foundVuln
		vulns = append(vulns, foundVuln.Vulnerability.VulnerabilityIDs[0].VulnerabilityID)
	}

	for vulnID, foundVex := range neighborVex {
		foundVex := foundVex
		if foundVex.Status == model.VexStatusFixed || foundVex.Status == model.VexStatusNotAffected {
			delete(neighborVulns, vulnID)
		}
	}

	return vulns
}
