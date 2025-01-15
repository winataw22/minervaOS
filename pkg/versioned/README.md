# versioned module

The `versioned` package provides utilities to manage versioned data streams and files. It uses the `blang/semver` package to handle semantic versioning.

## Error Handling

`ErrNotVersioned`: Error raised if the underlying reader has no version information.

`IsNotVersioned`: A function checks if an error is caused by a "not versioned" stream.

## Structs

`Reader`: Represents a versioned reader that can load the version of the data from a stream.

### Fields

- `Reader io.Reader`: The underlying data stream.
- `version Version`: The version of the data.

### Methods

`Version`: Returns the version of the data.

## Functions

`NewVersionedReader`: Creates a `Reader` with a specified version.

`NewReader`: Creates a `Reader` by reading the version from a stream. Fails if the stream lacks version information.

`ReadFile`: Reads the content of a file and its version.

`NewWriter`: Creates a versioned writer that marks data with the provided version.

`WriteFile`: Writes versioned data to a file.

`Parse`: Parses a version string into a `Version` object.

`MustParse`: Parses a version string into a `Version` object. Panics if the parsing fails.

`ParseRange`: Parses a version range string into a `Range` object.

`MustParseRange`: Parses a version range string into a `Range` object. Panics if the parsing fails.
