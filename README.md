# bumptag
[![GitHub release](https://img.shields.io/github/release/SVilgelm/bumptag.svg)](https://GitHub.com/SVilgelm/bumptag/releases/)
![](https://github.com/SVilgelm/bumptag/workflows/Tests/badge.svg)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=SVilgelm_bumptag&metric=alert_status)](https://sonarcloud.io/dashboard?id=SVilgelm_bumptag)
[![GitHub license](https://img.shields.io/github/license/SVilgelm/bumptag.svg)](https://github.com/SVilgelm/bumptag/blob/master/LICENSE)

bumptag is a tool to increment a version and to create a git tag with an annotation

## Installation

### Using `brew`
```bash
brew install SVilgelm/apps/bumptag
```

### Go way
To install latest master:
```bash
go get github.com/SVilgelm/bumptag
```

### Binary mode

1. Check the [release](https://github.com/SVilgelm/bumptag/releases)
or the [latest release](https://github.com/SVilgelm/bumptag/releases/latest).
2. Download a build for your OS.

All builds are signed by gpg key [bumptag@vilgelm.info (17F40802)](http://hkps.pool.sks-keyservers.net/pks/lookup?search=bumptag%40vilgelm.info&fingerprint=on&op=index)


## Usage

```
$ bumptag --help
Usage: bumptag [<tagname>]

    <tagname>       The name of the tag to create, must be Semantic Versions 2.0.0 (http://semver.org)
    -e, --edit      Edit an annotation
    -r, --dry-run   Prints an annotation for the new tag
    -s, --silent    Do not show the created tag
    -a, --auto-push Push the created tag automatically
    -m, --major     Increment the MAJOR version
    -n, --minor     Increment the MINOR version (default)
    -p, --patch     Increment the PATCH version
        --version   Show a version of the bumptag tool
        --find-tag  Show the last tag, can be useful for CI tools
```

The script generates an annotation with all commits merged since the last tag.

To push new tag use:
```bash
$ git push origin --tags
```

Or set `--auto-push` flag


### Examples:

* ```$ bumptag``` creates a tag with +1 for minor (v1.0.0 -> v1.1.0)
* ```$ bumptag -p``` increment PATCH version (v1.0.0 -> v1.0.1), for bug fixes
* ```$ bumptag v2.10.4``` creates the v2.10.4 tag
* ```$ bumptag --auto-push v2.10.4``` creates the v2.10.4 tag and pushes it to a remote
* ```$ bumptag --edit v2.10.4 ``` creates the v2.10.4 tag and runs an editor to manually edit the annotation

#### Simple scenario:
Preparing a first release of `bumptag` tool
```bash
$ go run bumptag.go
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

#### Complex scenario
A repo with a branch for specific version
```bash
# Initializing an empty git repo
07:37:20$ mkdir test
07:37:35$ cd test
07:37:39$ git init
Initialized empty Git repository in /tmp/test/.git/

# Creating first commit and tagging it
07:37:44$ touch foo
07:38:02$ git add foo
07:38:10$ git commit -m"foo"
[master (root-commit) f4cefb4] foo
 1 file changed, 0 insertions(+), 0 deletions(-)
 create mode 100644 foo
07:38:22$ bumptag
tag v0.1.0
Tagger: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:40:54 2019 -0500

Bump version v0.1.0

* f4cefb4 foo

commit f4cefb4599eed591c418f2b1e2667d555d006086
Author: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:38:22 2019 -0500

    foo

diff --git a/foo b/foo
new file mode 100644
index 0000000..e69de29

07:40:54$ git log
commit f4cefb4599eed591c418f2b1e2667d555d006086 (HEAD -> master, tag: v0.1.0)
Author: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:38:22 2019 -0500

    foo

# Preparing branch v0
07:41:06$ git checkout -b v0
Switched to a new branch 'v0'

# Developing master branch to version 1
07:41:32$ git checkout master
Switched to branch 'master'
07:41:44$ touch bar
07:41:57$ git add bar
07:42:02$ git commit -m"bar"
[master 11c1819] bar
 1 file changed, 0 insertions(+), 0 deletions(-)
 create mode 100644 bar

07:42:46$ bumptag -m
tag v1.0.0
Tagger: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:42:50 2019 -0500

Bump version v1.0.0

* 11c1819 bar

commit 11c1819845a995f6fb437d7af8134beffe7b588a
Author: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:42:11 2019 -0500

    bar

diff --git a/bar b/bar
new file mode 100644
index 0000000..e69de29
07:42:50$ git log
commit 11c1819845a995f6fb437d7af8134beffe7b588a (HEAD -> master, tag: v1.0.0)
Author: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:42:11 2019 -0500

    bar

commit f4cefb4599eed591c418f2b1e2667d555d006086 (tag: v0.1.0, v0)
Author: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:38:22 2019 -0500

    foo

# Patching v0 branch and tagging
07:42:53$ git checkout v0
Switched to branch 'v0'
07:42:59$ touch xyz
07:43:08$ git add xyz
07:43:15$ git commit -m"xyz"
[v0 3999337] xyz
 1 file changed, 0 insertions(+), 0 deletions(-)
 create mode 100644 xyz
07:43:22$ bumptag -p
tag v0.1.1
Tagger: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:43:30 2019 -0500

Bump version v0.1.1

* 3999337 xyz

commit 39993375898d1e5c9cbc03b22a20efb8f67762d4
Author: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:43:22 2019 -0500

    xyz

diff --git a/xyz b/xyz
new file mode 100644
index 0000000..e69de29
07:43:30$ git log
commit 39993375898d1e5c9cbc03b22a20efb8f67762d4 (HEAD -> v0, tag: v0.1.1)
Author: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:43:22 2019 -0500

    xyz

commit f4cefb4599eed591c418f2b1e2667d555d006086 (tag: v0.1.0)
Author: Sergey Vilgelm <sergey@vilgelm.info>
Date:   Mon Jun 10 07:38:22 2019 -0500

    foo
```

## License

MIT licensed. See the bundled [LICENSE](LICENSE) file for more details.
