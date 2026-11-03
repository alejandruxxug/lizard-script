-- Closures: make a counter and call it a million times.
local function counter()
  local n = 0
  return function()
    n = n + 1
    return n
  end
end
local next = counter()
local last = 0
local i = 0
while i < 1000000 do
  last = next()
  i = i + 1
end
print(last)
