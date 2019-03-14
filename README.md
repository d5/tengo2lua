# Tengo2Lua 

An experimental transpiler to convert [Tengo](https://github.com/d5/tengo) source code into Lua code. 

### Limitations

- Tengo module system is not implemented.
- Tengo `int` and `float` values are both converted into `Number` values in Lua.
- Array slicing is not implemented (`[1,2,3,4,5][1:3]`). _But string slicing is implemented._
- Character literal is not supported.
- String indexing is not supported.
- Bitwise operators is not implemented. 
- Different boolean truthiness
- Different type coercion logic

### Example

Tengo code:

```golang
each := func(x, f) { for k, v in x { f(k, v) } }
sum := 0
each([1, 2, 3], func(i, v) { sum += v })
```

is converted into:

```lua
function __iter__(v)
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
end
getmetatable("").__add=function(a,b) return a..b end
local each=function(x,f)
  for k, v in __iter__(x) do
    local __cont_1__ = false
    repeat
      f(k,v)
      __cont_1__ = true
    until 1
    if not __cont_1__ then break end
  end
end

local sum=0
each({[0]=(1),(2),(3), __a=true},function(i,v)
  sum=sum+v
end
)
```
