-- Float math: 5 million rounds of a multiply and an add.
local function run(n)
  local x = 0.0
  local i = 0
  while i < n do
    x = x * 0.999 + 1.5
    i = i + 1
  end
  return x
end
print(run(5000000))
