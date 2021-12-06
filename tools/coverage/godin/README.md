== Measuring Code Coverage of Golang Binaries with Godin ==

Godin can generate and publish code coverage reports while the service is running!

You can test the service manually or automatically, whatever, you don't have to stop the tested service to get the 
coverage report anymore.

Godin is part of the Frisbee project, and is designed for some complex scenarios, like system testing code coverage 
collection and accurate testing.



== Why Godin ==

Measuring coverage of Go code is an easy task with the built-in go test tool, but for tests that run a binary,
like end-to-end or automated tests, there’s no obvious way to measure coverage.
Despite its merits, go test tool is only designed for unit test coverage collection, thus coming with the following deficiencies:
1) limited by go test command, we have to close the program to collect test coverage.
2) the code repository may be polluted, inconvenient for local develop.
3) flag injected, like -test.converprofile, -test.timeout and so on.


Considering the deficiencies above, We’ve open sourced a tool called Godin, which solves this problem by providing a 
simple Docker pipeline which takes as input the target repository, and generates an “instrumented binary” that can 
measure its own coverage, runs it with user-specified command line arguments and environment variables, and 
publish its coverage reports.  

Godin automates the following steps:

1) Download the source code for the target service
2) Use a tool like goc to generate runtime coverage reports.
3) Use a tool like gcov2lcov to convert golang test coverage to lcov format (which can for example be uploaded to coveralls).
4) Use a tool like gehtml to convert the lcov format into a comprehensive html phage.






