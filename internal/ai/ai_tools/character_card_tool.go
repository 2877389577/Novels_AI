package ai_tools

import (
	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

type CharacterCardTool struct {
	Name        string `json:"name" jsonschema:"required" jsonschema_description:"角色的名称"`
	Gender      string `json:"gender" jsonschema:"required" jsonschema_description:"角色的性别，只限【男、女】，不知道默认为：男"`
	Intro       string `json:"intro,omitempty" jsonschema_description:"角色的介绍"`
	Personality string `json:"personality,omitempty" jsonschema_description:"角色的性格特征"`
	Appearance  string `json:"appearance,omitempty" jsonschema_description:"角色的外观特征"`
	Ability     string `json:"ability,omitempty" jsonschema_description:"角色的能力特征"`
}

func GetCharacterCardToolSchema() (*schema.ToolInfo, error) {
	return utils.GoStruct2ToolInfo[CharacterCardTool]("character_card_tool", "角色卡片工具,用来创建角色卡片基础信息")
}

func ToolOutput2CharacterCardTool(blocks []*schema.ContentBlock) []*CharacterCardTool {
	if len(blocks) == 0 {
		return nil
	}
	cs := make([]*CharacterCardTool, 3)
	for _, block := range blocks {
		if block.Type == schema.ContentBlockTypeFunctionToolCall {
			var c CharacterCardTool
			err := sonic.Unmarshal([]byte(block.FunctionToolCall.Arguments), &c)
			if err == nil {
				cs = append(cs, &c)
			}
		}
	}
	return cs
}
