package toolsutil

import (
	"github.com/quailyquaily/mistermorph/llm"
	"github.com/quailyquaily/mistermorph/tools"
	"github.com/quailyquaily/mistermorph/tools/builtin"
)

func BindTodoUpdateToolLLM(reg *tools.Registry, client llm.Client, model string) {
	if reg == nil {
		return
	}
	raw, ok := reg.Get("todo_update")
	if !ok {
		return
	}
	tool, ok := raw.(*builtin.TodoUpdateTool)
	if !ok {
		return
	}
	tool.BindLLM(client, model)
}
