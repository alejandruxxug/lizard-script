-- Integer loop: 10 million rounds of a modulo and an addition.
local function run(n)
  local t = 0
  local i = 0
  while i < n do
    t = t + i % 7
    i = i + 1
  end
  return t
end
print(run(10000000))
