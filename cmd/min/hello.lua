--[[
local function foo()
    local a = 1009
    local function bar()
        local a = a
        print(a)
    end
    bar()
end
foo()]]

local a = 100
local b = 200
local c = a + b + 2 * a
print(c)