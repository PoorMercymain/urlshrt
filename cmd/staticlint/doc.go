// staticlint - custom multichecker for urlshrt project.
//
// To use it, write go build . in directory cmd/staticlint/ and after that you'll have a binary file staticlint .
//
// Then go to the root directory of the project and use command .\cmd\staticlint\staticlint.exe ./... .
//
// Analyzers used are: standard static analyzers of the analysis/passes package, all the analyzers of the SA class of the staticcheck package and S1020 check for redundant nil check in type assertion
//
// Also, I used two public analyzers  - faillint (to restrict import of some packages (the functionality is not used, but exist)) and errwrap (to fmt.Errorf correct error wrapping)
package main
