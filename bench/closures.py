# Closures: make a counter and call it a million times.
def counter():
    n = 0
    def next():
        nonlocal n
        n += 1
        return n
    return next

def main():
    nxt = counter()
    last = 0
    i = 0
    while i < 1000000:
        last = nxt()
        i += 1
    print(last)
main()
