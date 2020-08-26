def task(name, instance, env={}, instructions=[]):
    result = {
        'name': name,
        'env': env,
    }
    result.update(instance.items())
    for instruction in instructions:
        result.update(instruction.items())
    return result


def script(name, *lines):
    return {
        name + '_script': lines
    }


def artifacts(name, path, type="", format=""):
    result = {
        'path': path,
    }
    if type != "":
        result['type'] = type
    if format != "":
        result['format'] = format
    return {
        name + '_artifacts': result
    }


def container(image="", dockerfile="", cpu=2.0, memory=4096):
    result = dict()
    if image != "":
        result['image'] = image
    if dockerfile != "":
        result['dockerfile'] = dockerfile
    result['cpu'] = cpu
    result['memory'] = memory
    return {
        'container': result
    }


def always(instruction):
    return {'always': instruction}


def on_failure(instruction):
    return {'on_failure': instruction}


def on_sucess(instruction):
    return {'on_sucess': instruction}
