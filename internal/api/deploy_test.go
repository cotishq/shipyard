package api

import "testing"

func TestValidateDeployRequest_AllowsStaticCopyFromGitHub(t *testing.T) {
	req := &ProjectCreateRequest{
		RepoURL:     "https://github.com/mdn/beginner-html-site",
		BuildPreset: "static-copy",
		OutputDir:   "",
	}

	buildCommand, err := validateDeployRequest(req)
	if err != nil {
		t.Fatalf("expected request to be valid, got error: %v", err)
	}

	if buildCommand != "true" {
		t.Fatalf("expected static-copy build command, got %q", buildCommand)
	}
}

func TestValidateDeployRequest_RejectsDisallowedHost(t *testing.T) {
	req := &ProjectCreateRequest{
		RepoURL:     "https://gitlab.com/example/project",
		BuildPreset: "static-copy",
		OutputDir:   "",
	}

	_, err := validateDeployRequest(req)
	if err == nil || err.Error() != "repo host is not allowed" {
		t.Fatalf("expected disallowed host error, got %v", err)
	}
}

func TestValidateDeployRequest_RejectsUnsupportedPreset(t *testing.T) {
	req := &ProjectCreateRequest{
		RepoURL:     "https://github.com/mdn/beginner-html-site",
		BuildPreset: "custom",
		OutputDir:   "dist",
	}

	_, err := validateDeployRequest(req)
	if err == nil || err.Error() != "unsupported build_preset" {
		t.Fatalf("expected unsupported preset error, got %v", err)
	}
}

func TestValidateDeployRequest_RejectsPathTraversal(t *testing.T) {
	req := &ProjectCreateRequest{
		RepoURL:     "https://github.com/mdn/beginner-html-site",
		BuildPreset: "vite",
		OutputDir:   "../dist",
	}

	_, err := validateDeployRequest(req)
	if err == nil || err.Error() != "output_dir must not escape repository root" {
		t.Fatalf("expected path traversal error, got %v", err)
	}
}

func TestValidateDeployRequest_NormalizesDotOutputDir(t *testing.T) {
	req := &ProjectCreateRequest{
		RepoURL:     "https://github.com/mdn/beginner-html-site",
		BuildPreset: "vite",
		OutputDir:   ".",
	}

	buildCommand, err := validateDeployRequest(req)
	if err != nil {
		t.Fatalf("expected request to be valid, got error: %v", err)
	}

	if buildCommand == "" {
		t.Fatal("expected build command to be resolved")
	}

	if req.OutputDir != "" {
		t.Fatalf("expected output dir to normalize to empty string, got %q", req.OutputDir)
	}
}
