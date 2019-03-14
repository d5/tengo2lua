package tengo2lua

type helper int

const (
	helperStringConcat helper = iota
	helperIterator
	helperSlicing
)

var helpers = map[helper]string{
	// string concatenation operator
	helperStringConcat: `getmetatable("").__add=function(a,b) return a..b end`,
	// iterator
	helperIterator: `function __iter__(v)
    	if v.__a then
        	local idx = 0
	        return function()
    	        if v[idx] == nil then
        	        return nil
            	end
            	idx = idx + 1
            	return idx-1, v[idx-1]
        	end
    	else
        	return pairs(v)
    	end
	end`,
	// slicing operator
	helperSlicing: `function __slice__(v, l, h)
    	if v.__a then
			if l >= h then return {[0]=nil, __a=true} end
			return {[0]=nil, __a=true}
    	else
        	return string.sub(v, l+1, h)
    	end
	end`,
}
