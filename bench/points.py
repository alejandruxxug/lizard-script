# Point math: 2 million additions of 2D points (as tuples).
def run(n):
    p = (0.0, 0.0)
    i = 0
    while i < n:
        p = (p[0] + 1.0, p[1] + 2.0)
        i += 1
    return p
print(run(2000000))
