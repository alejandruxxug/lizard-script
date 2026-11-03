# Strings: build a 200,000-character string piece by piece, then count a letter.
def run(n):
    s = ""
    i = 0
    while i < n:
        s += "ab"
        i += 1
    count = 0
    i = 0
    while i < len(s):
        if s[i] == "a":
            count += 1
        i += 1
    return count
print(run(100000))
