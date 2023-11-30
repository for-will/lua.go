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
--[[
local a = 100
local b = 200
local function foo()
    return a + b
end
print(foo())
--]]

--[[
local a = 1
local b = a + 4 + 5
]]

--- 关系运算符
--[[
local a, b, c = 1, 2, 3
a = b == c
print(b)
]]

--- 分支

--[[
local a = 3
local x = 0
if a ~= 1 then
    x = 100
elseif a == 2 then
    x = 200
elseif a == 3 then
    x = 300
else
    x = -1
end
print(x)

--]]

--- 逻辑指令
--[[local a = 'what'
local b = 'YES'
local c = 'All Right!'
local x = a and b or c]]
--print(x)
--if a then
--    x = 100
--else
--    x = 200
--end

--- 循环指令
--[[
local a = 100
for _ = 10, 3, -1 do
    print(_)
end]]

--- 泛型循环
local t = { 1, 2, 3, 4 }
for k, v in pairs(t) do
    print(v)
end
