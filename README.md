# bumpversion
[![GitHub release](https://img.shields.io/github/release/SVilgelm/bumpversion.svg)](https://GitHub.com/SVilgelm/bumpversion/releases/)
[![Build Status](https://travis-ci.com/SVilgelm/bumpversion.svg?branch=master)](https://travis-ci.com/SVilgelm/bumpversion)
[![Go Report Card](https://goreportcard.com/badge/github.com/SVilgelm/bumpversion)](https://goreportcard.com/report/github.com/SVilgelm/bumpversion)
[![GitHub license](https://img.shields.io/github/license/SVilgelm/bumpversion.svg)](https://github.com/SVilgelm/bumpversion/blob/master/LICENSE)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2FSVilgelm%2Fbumpversion.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2FSVilgelm%2Fbumpversion?ref=badge_shield)

bumpversion is a tool to increment a version and to create a git tag with an annotation

## Installation

To install latest master:
```go
go get github.com/SVilgelm/bumpversion
```

Or download an executable file: https://github.com/SVilgelm/bumpversion/releases/latest

## Usage

```
$ bumpversion --help
Usage: bumpversion [<tagname>]

    <tagname>       The name of the tag to create, must be Semantic Versions 2.0.0 (http://semver.org)
    -r, --dry-run   Prints an annotation for the new tag
    -m, --major     Increment the MAJOR version
    -n, --minor     Increment the MINOR version (default)
    -p, --pathc     Increment the PATCH version
        --version   Show a version of the bumpversion tool
        --find-tag  Show the last tag, can be useful for CI tools
```

### Examples:

* ```$ bumpversion``` creates a tag with +1 for minor (v1.0.0 -> v1.1.0)
* ```$ bumpversion -p``` increment PATCH version (v1.0.0 -> v1.0.1), for bug fixes
* ```$ bumpversion v2.10.4``` creates the v2.10.4 tag

```bash
$ go run bumpversion.go
tag v0.1.0
Tagger: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Fri Jun 7 12:28:02 2019 -0500

Bump version v0.1.0

* b0e6bb9 Add Makefile
* a1f5363 Update README.md
* 88b167f Upload binaries
* 0358b2a Bumpversion
* 99e339a Integration with travis-ci
* dd4e378 Add LICENSE section in README.md
* d2db2d8 Initial commit
-----BEGIN PGP SIGNATURE-----
...
-----END PGP SIGNATURE-----

commit b0e6bb9f3c249a8e61365422ab84d8d836af8ac5
Author: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Fri Jun 7 12:10:35 2019 -0500
...
```

The script also generates an annotation with all commits merged since the last tag.

And don't forget to execute:

```bash
$ git push origin --tags
```

## License

MIT licensed. See the bundled [LICENSE](LICENSE) file for more details.
