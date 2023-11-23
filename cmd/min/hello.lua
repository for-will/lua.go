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

--[[
local a = 100
local b = 200
local c = a + b + 2 * a
print(c)
--]]



--local a = { 4, 5, 6 }
--print(a[1])

--[[a = 200
local b = 100
function foo()
    b = b + a
end

foo()]]

--local id = 0
--a = 100
--local function next()
--    id = id + 1
--    return a * id
--end
--
--next()


--[[

local function nums()
    return 1, 2, 3
end
--print(nums(), nums())
local a, b = nums()
local c, d, e = nums()
--local d, e, f = nums()
--print(a, b, c, d, e, f)
]]

--- self 相关指令
--[[

local foo = { prompt = 'say: ' }

function foo:bar(v)
    print(self.prompt .. v)
end

foo:bar('Hello!!')
foo.bar(foo, "How do you do?")
]]

--- OP_CLOSURE 设置upvalue
local a = 100
local b = 200
local function foo()
    return a + b
end
print(foo())