# Release Process

This document outlines the process for creating new releases of Railpack.

## Creating a New Release

1. Determine the new version number following [Semantic
   Versioning](https://semver.org/) principles:

   - MAJOR version for incompatible API changes
   - MINOR version for backwards-compatible functionality additions
   - PATCH version for backwards-compatible bug fixes

2. Create and push a new tag with the version number:

   ```bash
   git tag v1.2.3  # Replace with your version number
   git push origin --tags
   ```

3. The [release
   workflow](https://github.com/salamer/railpack/actions/workflows/release.yml)
   will automatically:
   - Build and publish the frontend Docker image to
     [GHCR](https://github.com/orgs/railwayapp/packages?repo_name=railpack)
   - Create a GitHub release with changelog
   - Build and attach binaries for multiple platforms

## Release Artifacts

### Frontend Docker Image

The frontend Docker image is published to GitHub Container Registry (GHCR) with
the following tags:

- `ghcr.io/railpack/railpack-frontend:latest` (on default branch)
- `ghcr.io/railpack/railpack-frontend:v1.2.3` (specific version)
- `ghcr.io/railpack/railpack-frontend:1.2` (minor version)

The image is built for both `linux/amd64` and `linux/arm64` platforms.

### Binary Releases

The release workflow automatically builds and attaches binaries for multiple
platforms to the GitHub release.

## Verifying a Release

After pushing a tag:

1. Check the [Actions tab](https://github.com/salamer/railpack/actions) to
   monitor the release workflow
2. Verify the [GitHub release](https://github.com/salamer/railpack/releases)
   is created with the correct artifacts
3. Confirm the frontend Docker image is available in the [package
   registry](https://github.com/salamer/railpack/pkgs/container/railpack-frontend)
