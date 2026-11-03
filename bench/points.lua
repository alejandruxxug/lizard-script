-- Point math: 2 million additions of 2D points (as two-element tables).
local function run(n)
  local p = {0.0, 0.0}
  local i = 0
  while i < n do
    p = {p[1] + 1.0, p[2] + 2.0}
    i = i + 1
  end
  return p
end
local p = run(2000000)
print("(" .. p[1] .. ", " .. p[2] .. ")")
