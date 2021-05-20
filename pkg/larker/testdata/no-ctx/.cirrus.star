def main():
    return [{'script': 'uptime'}]

def on_build_created():
    print("it works fine without ctx argument!")
