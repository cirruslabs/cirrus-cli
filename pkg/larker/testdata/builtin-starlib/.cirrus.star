load("cirrus", "http", "hash", "base64", "json", "yaml", "re")
load("cirrus", "zipfile", "fs")

def main(ctx):
    test_http()
    test_hash()
    test_base64()
    test_json()
    test_yaml()
    test_re()
    test_zipfile()

    return []

def test_http():
    resp = http.get("https://httpbin.org/json")
    if resp.status_code != 200 or resp.json().get("slideshow") == None:
        fail("failed to parse JSON")

    resp = http.post("https://httpbin.org/status/418")
    if resp.status_code != 418:
        fail("HTTP POST returned status code %s instead of 418" % http.status_code)

    resp = http.put("https://httpbin.org/status/418")
    if resp.status_code != 418:
        fail("HTTP PUT returned status code %s instead of 418" % http.status_code)

    resp = http.patch("https://httpbin.org/status/418")
    if resp.status_code != 418:
        fail("HTTP PATCH returned status code %s instead of 418" % http.status_code)

    resp = http.delete("https://httpbin.org/status/418")
    if resp.status_code != 418:
        fail("HTTP DELETE returned status code %s instead of 418" % http.status_code)

    resp = http.options("https://httpbin.org/")
    allow_header = resp.headers.get("Allow")
    if allow_header == None or "OPTIONS" not in allow_header:
        fail("Allow header does not contain OPTIONS method: %s" % allow_header)

def test_hash():
    test_vector = "The quick brown fox jumps over the lazy dog"

    md5_result = hash.md5(test_vector)
    if md5_result != "9e107d9d372bb6826bd81d3542a419d6":
        fail("MD5(%s) returned unexpected value %s" % (test_vector, md5_result))

    sha1_result = hash.sha1(test_vector)
    if sha1_result != "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12":
        fail("SHA-1(%s) returned unexpected value %s" % (test_vector, sha1_result))

    sha256_result = hash.sha256(test_vector)
    if sha256_result != "d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592":
        fail("SHA-256(%s) returned unexpected value %s" % (test_vector, sha256_result))

def test_base64():
    plain = "foob"
    encoded = "Zm9vYg=="

    encode_result = base64.encode(plain)
    if encode_result != encoded:
        fail("base64 encoded %s into %s, but expected %s" % (plain, encode_result, encoded))

    decode_result = base64.decode(encoded)
    if decode_result != plain:
        fail("base64 decoded %s into %s, but expected %s" % (encoded, decode_result, plain))

def test_json():
    python_obj = {"key": 42}
    json_obj = "{\"key\":42}"

    marshalled = json.dumps(python_obj)
    if marshalled != json_obj:
        fail("json marshalling failed, expected '%s', got '%s'" % (json_obj, marshalled))

    unmarshalled = yaml.loads(json_obj)
    if unmarshalled != python_obj:
        fail("json unmarshalling failed, expected '%s', got '%s'" % (python_obj, unmarshalled))

def test_yaml():
    python_obj = {"key": 42}
    yaml_obj = "key: 42\n"

    marshalled = yaml.dumps(python_obj)
    if marshalled != yaml_obj:
        fail("yaml marshalling failed, expected '%s', got '%s'" % (yaml_obj, marshalled))

    unmarshalled = yaml.loads(yaml_obj)
    if unmarshalled != python_obj:
        fail("yaml unmarshalling failed, expected '%s', got '%s'" % (python_obj, unmarshalled))

def test_re():
    findall_expected = ("AAA", "BB")
    findall_actual = re.findall("[ABC]{2,}", "AAAzzzBBzzzC")
    if findall_actual != findall_expected:
        fail("re.findall() returned %s instead of %s" % (findall_actual, findall_expected))

    split_expected = ("abc", "def", "ghi")
    split_actual = re.split("[^a-z]", "abc def\tghi")
    if split_actual != split_expected:
        fail("re.split() returned %s instead of %s" % (split_actual, split_expected))

    sub_expected = "[snip][snip][snip]ABC"
    sub_actual = re.sub("[0-9]", "[snip]", "123ABC")
    if sub_actual != sub_expected:
        fail("re.sub() returned %s instead of %s" % (sub_actual, sub_expected))

def test_zipfile():
    zf = zipfile.ZipFile(fs.read("test.zip"))

    namelist_expected = ["test.txt"]
    namelist_actual = zf.namelist()
    if namelist_actual != namelist_expected:
        fail("zf.namelist() returned %s instead of %s" % (namelist_actual, namelist_expected))

    read_expected = "test\n"
    read_actual = zf.open("test.txt").read()
    if read_actual != read_expected:
        fail("ZipInfo.read() returned %s instead of %s" % (read_actual, read_expected))
