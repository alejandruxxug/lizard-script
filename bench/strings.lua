-- Strings: build a 200,000-character string piece by piece, then count a letter.
local function run(n)
  local s = ""
  local i = 0
  while i < n do
    s = s .. "ab"
    i = i + 1
  end
  local count = 0
  i = 1
  while i <= #s do
    if string.sub(s, i, i) == "a" then count = count + 1 end
    i = i + 1
  end
  return count
end
print(run(100000))
