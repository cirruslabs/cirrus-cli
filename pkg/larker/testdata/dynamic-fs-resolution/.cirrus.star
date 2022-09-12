load("cirrus", "fs")

def main():
    if fs.exists("github.com/cirruslabs/tart/Package.swift@master"):
        fail("Package.swift does not exist!")

    return []
