package workflow

// OpenCodeEngine is the behavior-defined OpenCode CLI engine.
type OpenCodeEngine struct {
	*BehaviorDefinedEngine
}

func NewOpenCodeEngine() *OpenCodeEngine {
	def, err := getBuiltinEngineDefinition("opencode")
	if err != nil {
		panic(err)
	}
	engine, err := NewBehaviorDefinedEngine(def)
	if err != nil {
		panic(err)
	}
	return &OpenCodeEngine{BehaviorDefinedEngine: engine}
}
