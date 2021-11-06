load("cirrus", "fs")

shouldExist = "exists-should-exist.txt"
someFile = "read-some-file.txt"

def main(ctx):
    test_exists()
    test_read()
    test_readdir()
    test_isdir()

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
        "dir",
        shouldExist,
        someFile,
    ]
    actualFiles = fs.readdir(".")

    if expectedFiles != actualFiles:
        fail("directory contains %s instead of %s" % (actualFiles, expectedFiles))

    shouldNotExist = "readdir-should-not-exist"
    if fs.readdir(shouldNotExist) != None:
        fail("non-existent directory %s should not be readable" % shouldNotExist)

def test_isdir():
    file = "dir/file"
    dir = "dir"

    if fs.isdir(file):
        fail("fs.isdir() reports that the file we've created is a directory")

    if not fs.isdir(dir):
        fail("fs.isdir() reports that the directory we've created is not a directory")

    if fs.isdir("does-not-exist-really") != None:
        fail("fs.isdir() should return None on non-existent path")
