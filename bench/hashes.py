# Hashes: build 200,000 small records and sum one field.
def run(n):
    rows = []
    i = 0
    while i < n:
        rows.append({"id": i, "x": i * 2, "name": "row"})
        i += 1
    total = 0
    i = 0
    while i < len(rows):
        total += rows[i]["x"]
        i += 1
    return total
print(run(200000))
