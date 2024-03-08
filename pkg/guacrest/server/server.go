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
	"sync"

	"github.com/Khan/genqlient/graphql"
	model "github.com/guacsec/guac/pkg/assembler/clients/generated"
	"github.com/guacsec/guac/pkg/assembler/helpers"
	gen "github.com/guacsec/guac/pkg/guacrest/generated"
	"github.com/guacsec/guac/pkg/logging"
	"github.com/guacsec/guac/pkg/misc/depversion"
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
}

func NewDefaultServer(gqlClient graphql.Client) *DefaultServer {
	return &DefaultServer{gqlClient: gqlClient}
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

func (s *DefaultServer) GetPackageSbomInformation(ctx context.Context, request gen.GetPackageSbomInformationRequestObject) (gen.GetPackageSbomInformationResponseObject, error) {
	pkgInput, err := helpers.PurlToPkg(request.Purl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PURL: %w", err)
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
	pkgResponse, err := model.Packages(ctx, s.gqlClient, *pkgFilter)
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
	pkgInput, err := helpers.PurlToPkg(request.Purl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PURL: %w", err)
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
	pkgResponse, err := model.Packages(ctx, s.gqlClient, *pkgFilter)
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

	var foundSlsaList []gen.Slsa

	for _, neighbor := range neighborResponse.Neighbors {
		switch v := neighbor.(type) {
		case *model.NeighborsNeighborsHasSLSA:
			foundSlsaList = append(foundSlsaList, v.Slsa.Origin)
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
	pkgInput, err := helpers.PurlToPkg(request.Purl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PURL: %w", err)
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
	pkgResponse, err := model.Packages(ctx, s.gqlClient, *pkgFilter)
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

	var foundVulnerabilities []gen.Vulnerability

	for _, neighbor := range neighborResponse.Neighbors {
		switch v := neighbor.(type) {
		case *model.NeighborsNeighborsCertifyVuln:
			foundVulnerabilities = append(foundVulnerabilities, v.Vulnerability.VulnerabilityIDs[0].VulnerabilityID)
		// case *model.NeighborsNeighborsCertifyBad:
		// 	collectedNeighbors.badLinks = append(collectedNeighbors.badLinks, v)
		// 	path = append(path, v.Id)
		// case *model.NeighborsNeighborsCertifyGood:
		// 	collectedNeighbors.goodLinks = append(collectedNeighbors.goodLinks, v)
		// 	path = append(path, v.Id)
		// case *model.NeighborsNeighborsCertifyScorecard:
		// 	collectedNeighbors.scorecards = append(collectedNeighbors.scorecards, v)
		// 	path = append(path, v.Id)
		// case *model.NeighborsNeighborsCertifyVEXStatement:
		// 	collectedNeighbors.vexLinks = append(collectedNeighbors.vexLinks, v)
		// 	path = append(path, v.Id)
		default:
			continue
		}
	}

	vulnsList, err := searchPkgViaHasSBOM(ctx, s.gqlClient, request.Purl, 0, true)
	if err != nil {
		fmt.Print("fail")
	}

	foundVulnerabilities = append(foundVulnerabilities, vulnsList...)

	val := gen.GetPackageVulnInformation200JSONResponse{
		VulnInfoJSONResponse: gen.VulnInfoJSONResponse{
			Vulnerabilities: foundVulnerabilities,
		},
	}

	return val, nil
}

func (s *DefaultServer) GetPackageInformation(ctx context.Context, request gen.GetPackageInformationRequestObject) (gen.GetPackageInformationResponseObject, error) {
	pkgInput, err := helpers.PurlToPkg(request.Purl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PURL: %w", err)
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
	pkgResponse, err := model.Packages(ctx, s.gqlClient, *pkgFilter)
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
		// case *model.NeighborsNeighborsCertifyBad:
		// 	collectedNeighbors.badLinks = append(collectedNeighbors.badLinks, v)
		// 	path = append(path, v.Id)
		// case *model.NeighborsNeighborsCertifyGood:
		// 	collectedNeighbors.goodLinks = append(collectedNeighbors.goodLinks, v)
		// 	path = append(path, v.Id)
		// case *model.NeighborsNeighborsCertifyScorecard:
		// 	collectedNeighbors.scorecards = append(collectedNeighbors.scorecards, v)
		// 	path = append(path, v.Id)
		// case *model.NeighborsNeighborsCertifyVEXStatement:
		// 	collectedNeighbors.vexLinks = append(collectedNeighbors.vexLinks, v)
		// 	path = append(path, v.Id)
		case *model.NeighborsNeighborsHasSBOM:
			foundSbomList = append(foundSbomList, v.DownloadLocation)
		case *model.NeighborsNeighborsHasSLSA:
			foundSlsaList = append(foundSlsaList, v.Slsa.Origin)
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

	vulnsList, err := searchPkgViaHasSBOM(ctx, s.gqlClient, request.Purl, 0, true)
	if err != nil {
		fmt.Print("fail")
	}

	foundVulnerabilities = append(foundVulnerabilities, vulnsList...)

	val := gen.GetPackageInformation200JSONResponse{
		PurlInfoJSONResponse: gen.PurlInfoJSONResponse{
			SbomList:        foundSbomList,
			SlsaList:        foundSlsaList,
			Vulnerabilities: foundVulnerabilities,
		},
	}

	return val, nil
}

func concurrentVulnAndVexNeighbors(ctx context.Context, gqlclient graphql.Client, pkgID string, isDep model.AllHasSBOMTreeIncludedDependenciesIsDependency, resultChan chan<- struct {
	pkgVersionNeighborResponse *model.NeighborsResponse
	isDep                      model.AllHasSBOMTreeIncludedDependenciesIsDependency
}, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := logging.FromContext(ctx)
	pkgVersionNeighborResponse, err := model.Neighbors(ctx, gqlclient, pkgID, []model.Edge{model.EdgePackageCertifyVuln, model.EdgePackageCertifyVexStatement})
	if err != nil {
		logger.Errorf("error querying neighbor for vulnerability: %w", err)
		return
	}

	// Send the results to the resultChan
	resultChan <- struct {
		pkgVersionNeighborResponse *model.NeighborsResponse
		isDep                      model.AllHasSBOMTreeIncludedDependenciesIsDependency
	}{pkgVersionNeighborResponse, isDep}
}

// searchPkgViaHasSBOM takes in either a purl or URI for the initial value to find the hasSBOM node.
// From there is recursively searches through all the dependencies to determine if it contains hasSBOM nodes.
// It concurrent checks the package version node if it contains vulnerabilities and VEX data.
func searchPkgViaHasSBOM(ctx context.Context, gqlclient graphql.Client, searchString string, maxLength int, isPurl bool) ([]string, error) {
	checkedPkgIDs := make(map[string]bool)
	var foundVulns []string

	var wg sync.WaitGroup

	queue := make([]string, 0) // the queue of nodes in bfs
	type dfsNode struct {
		expanded bool // true once all node neighbors are added to queue
		parent   string
		pkgID    string
		depth    int
	}
	nodeMap := map[string]dfsNode{}

	nodeMap[searchString] = dfsNode{}
	queue = append(queue, searchString)

	resultChan := make(chan struct {
		pkgVersionNeighborResponse *model.NeighborsResponse
		isDep                      model.AllHasSBOMTreeIncludedDependenciesIsDependency
	})

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
				pkgResponse, err := getPkgResponseFromPurl(ctx, gqlclient, now)
				if err != nil {
					return nil, fmt.Errorf("getPkgResponseFromPurl - error: %v", err)
				}
				foundHasSBOMPkg, err = model.HasSBOMs(ctx, gqlclient, model.HasSBOMSpec{Subject: &model.PackageOrArtifactSpec{Package: &model.PkgSpec{Id: &pkgResponse.Packages[0].Namespaces[0].Names[0].Versions[0].Id}}})
				if err != nil {
					return nil, fmt.Errorf("failed getting hasSBOM via purl: %s with error :%w", now, err)
				}
			} else {
				foundHasSBOMPkg, err = model.HasSBOMs(ctx, gqlclient, model.HasSBOMSpec{Uri: &now})
				if err != nil {
					return nil, fmt.Errorf("failed getting hasSBOM via URI: %s with error: %w", now, err)
				}
			}
		} else {
			foundHasSBOMPkg, err = model.HasSBOMs(ctx, gqlclient, model.HasSBOMSpec{Subject: &model.PackageOrArtifactSpec{Package: &model.PkgSpec{Id: &now}}})
			if err != nil {
				return nil, fmt.Errorf("failed getting hasSBOM via purl: %s with error :%w", now, err)
			}
		}

		for _, hasSBOM := range foundHasSBOMPkg.HasSBOM {
			if pkgResponse, ok := foundHasSBOMPkg.HasSBOM[0].Subject.(*model.AllHasSBOMTreeSubjectPackage); ok {
				if pkgResponse.Type != guacType {
					if !checkedPkgIDs[pkgResponse.Namespaces[0].Names[0].Versions[0].Id] {
						vulnPath, err := queryVulnsViaPackageNeighbors(ctx, gqlclient, pkgResponse.Namespaces[0].Names[0].Versions[0].Id)
						if err != nil {
							return nil, fmt.Errorf("error querying neighbor: %v", err)
						}

						foundVulns = append(foundVulns, vulnPath...)
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
						return nil, fmt.Errorf("error from findMatchingDepPkgVersionIDs:%w", err)
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
					wg.Add(1)
					go concurrentVulnAndVexNeighbors(ctx, gqlclient, pkgID, isDep, resultChan, &wg)
					checkedPkgIDs[pkgID] = true
				}
			}
		}
		nowNode.expanded = true
		nodeMap[now] = nowNode
	}

	// Close the result channel once all goroutines are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	checkedCertifyVulnIDs := make(map[string]bool)

	// Collect results from the channel
	for result := range resultChan {
		neighborVulns := make(map[string]*model.NeighborsNeighborsCertifyVuln)
		neighborVex := make(map[string]*model.NeighborsNeighborsCertifyVEXStatement)

		for _, neighbor := range result.pkgVersionNeighborResponse.Neighbors {
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
		}

		foundVulns = append(foundVulns, removeVulnsWithValidVex(neighborVex, neighborVulns)...)

	}
	return foundVulns, nil
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

func queryVulnsViaPackageNeighbors(ctx context.Context, gqlclient graphql.Client, pkgVersionID string) ([]string, error) {
	neighborVulns := make(map[string]*model.NeighborsNeighborsCertifyVuln)
	neighborVex := make(map[string]*model.NeighborsNeighborsCertifyVEXStatement)

	var edgeTypes = []model.Edge{model.EdgePackageCertifyVuln, model.EdgePackageCertifyVexStatement}

	pkgVersionNeighborResponse, err := model.Neighbors(ctx, gqlclient, pkgVersionID, edgeTypes)
	if err != nil {
		return nil, fmt.Errorf("error querying neighbor for vulnerability: %w", err)
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
	}
	if !certifyVulnFound {
		return nil, fmt.Errorf("error certify vulnerability node not found, incomplete data. Please ensure certifier has run by running guacone certifier osv")
	}

	return removeVulnsWithValidVex(neighborVex, neighborVulns), nil
}

func removeVulnsWithValidVex(neighborVex map[string]*model.NeighborsNeighborsCertifyVEXStatement, neighborVulns map[string]*model.NeighborsNeighborsCertifyVuln) []string {
	var vulns []string
	for vulnID, foundVex := range neighborVex {
		if foundVex.Status == model.VexStatusFixed || foundVex.Status == model.VexStatusNotAffected {
			delete(neighborVulns, vulnID)
		}
	}

	for _, foundVuln := range neighborVulns {
		vulns = append(vulns, foundVuln.Vulnerability.VulnerabilityIDs[0].VulnerabilityID)
	}
	return vulns
}
