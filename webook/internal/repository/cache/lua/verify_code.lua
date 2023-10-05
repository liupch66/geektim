local key = KEYS[1]
-- 用户输入的 code
local expectedCode = ARGV[1]
local code = redis.call("get", key)
local cntKey = key .. ":cnt"
local cnt = tonumber(redis.call("get", cntKey))
if cnt <= 0 then
    -- 说明用户一直输错
    return -1
elseif code == expectedCode then
    -- 输入正确
    return 0
else
    -- 用户手一抖输错了
    return -2
end