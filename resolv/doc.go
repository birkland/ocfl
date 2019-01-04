// Package resolv provides facilities for identifying and resolving OCFL entities.  The intent is that eventually
// when there are multiple OCFL drivers (file, index, s3, http, etc), the resolv package would be responsible for
// invoking the correct driver for reading/writing OCFL entities.
package resolv
