# Integer loop: 10 million rounds of a modulo and an addition.
def run(n):
    t = 0
    i = 0
    while i < n:
        t += i % 7
        i += 1
    return t
print(run(10000000))
