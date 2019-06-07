# bumpversion
[![Build Status](https://travis-ci.com/SVilgelm/bumpversion.svg?branch=master)](https://travis-ci.com/SVilgelm/bumpversion)
[![Go Report Card](https://goreportcard.com/badge/github.com/SVilgelm/bumpversion)](https://goreportcard.com/report/github.com/SVilgelm/bumpversion)
[![GitHub license](https://img.shields.io/github/license/SVilgelm/bumpversion.svg)](https://github.com/SVilgelm/bumpversion/blob/master/LICENSE)
[![GitHub release](https://img.shields.io/github/release/SVilgelm/bumpversion.svg)](https://GitHub.com/SVilgelm/bumpversion/releases/)

bumpversion is a tool to increment a version and to create a git tag with an annotation

## Installation

```go
go get github.com/SVilgelm/bumpversion
```

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

The script also generates an annotation with all commits merged since the last tag.

And don't forget to execute:

```bash
$ git push origin --tags
```

## License

MIT licensed. See the bundled [LICENSE](LICENSE) file for more details.
