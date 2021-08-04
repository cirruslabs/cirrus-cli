def main():
    return [
        ('container', {'image': 'debian:latest'}),
        ('task', {'name': 'task1', 'script': True}),
        task('task2', True),
        ('task', {'name': 'task3', 'script': True}),
        task('task4', True)
    ]

def task(name, script):
    return {'name': name, 'script': script}
