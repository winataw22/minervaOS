# version module

The `version` package provides some utilities to get and display the version of a system, including branch, revision, and the dirty flag status.

## Constants

The following are placeholders that are replaced during the build of the system.

- `Branch`: Represents the branch the system is running of.
- `Revision`: Represents the code revision.
- `Dirty`: Shows if the binary was built from a repo with uncommitted changes.

## Version Interface

The `Version` interface defines two methods for retrieving version information

- `Short() string`: Returns a short representation of the version.
- `String() string`: Returns a detailed representation of the version.

## Functions

### `Current()`

Returns the current version as a `Version` object

### `ShowAndExit(short bool)`

Prints the version information and exits the program

- If `short` is `true`, prints the short version.
- Otherwise, prints the detailed version.

### `Parse(v string)`

Parses a version string using regular expressions and extracts the version and revision. Returns an error if the string is invalid
