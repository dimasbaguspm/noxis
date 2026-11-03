// Package constants holds every universal value shared across the API: the
// version and mount point, the driver and capability names, the change
// lifecycle, the wire operations, the HTTP header and media-type spellings,
// the limits and the metric catalog.
//
// One concern per file: a value belongs to the file named after the thing it
// describes, so a reader looking for a header never scrolls past a driver.
package constants
