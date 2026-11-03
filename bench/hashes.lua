-- Hashes: build 200,000 small records and sum one field.
local function run(n)
  local rows = {}
  local i = 0
  while i < n do
    rows[#rows + 1] = {id = i, x = i * 2, name = "row"}
    i = i + 1
  end
  local total = 0
  i = 1
  while i <= #rows do
    total = total + rows[i].x
    i = i + 1
  end
  return total
end
print(run(200000))
