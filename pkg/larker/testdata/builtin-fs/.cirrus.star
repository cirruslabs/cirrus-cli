load("cirrus", "fs")

shouldExist = "exists-should-exist.txt"
someFile = "read-some-file.txt"

def main(ctx):
    test_exists()
    test_read()
    test_readdir()

    return []

def test_exists():
    if not fs.exists(shouldExist):
        fail("%s does not exist, but should" % shouldExist)

    shouldNotExist = "exists-should-not-exist.txt"
    if fs.exists(shouldNotExist):
        fail("file %s should not exist" % shouldNotExist)

    if not fs.exists("."):
        fail("current directory does not exist, but should")

def test_read():
    expectedContents = "some-contents\n"
    actualContents = fs.read(someFile)

    if expectedContents != actualContents:
        fail("%s contains '%s' instead of '%s'" % (someFile, actualContents, expectedContents))

    shouldNotExist = "read-should-not-exist.txt"
    if fs.read(shouldNotExist) != None:
        fail("non-existent file %s should not be readable" % shouldNotExist)

def test_readdir():
    expectedFiles = [
        ".cirrus.star",
        shouldExist,
        someFile,
    ]
    actualFiles = fs.readdir(".")

    if expectedFiles != actualFiles:
        fail("directory contains %s instead of %s" % (expectedFiles, actualFiles))

    shouldNotExist = "readdir-should-not-exist"
    if fs.readdir(shouldNotExist) != None:
        fail("non-existent directory %s should not be readable" % shouldNotExist)
