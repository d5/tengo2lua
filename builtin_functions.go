package tengo2lua

var builtinFunctions = map[string]string{
	"len": `function(v)
    	if v.__a then
        	if v[0] == nil then return 0 else return #v + 1 end
    	elseif type(v) == "string" then
			return string.len(v)
		else
        	__n = 0
        	for _ in pairs(v) do __n = __n + 1 end
        	return __n
    	end
	end`,
}
