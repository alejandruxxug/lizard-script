# Float math: 5 million rounds of a multiply and an add.
def run(n):
    x = 0.0
    i = 0
    while i < n:
        x = x * 0.999 + 1.5
        i += 1
    return x
print(run(5000000))
