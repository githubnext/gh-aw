package workflow

// CrushEngine is the behavior-defined Crush CLI engine.
type CrushEngine struct {
	*BehaviorDefinedEngine
}

func NewCrushEngine() *CrushEngine {
	def, err := getBuiltinEngineDefinition("crush")
	if err != nil {
		panic(err)
	}
	engine, err := NewBehaviorDefinedEngine(def)
	if err != nil {
		panic(err)
	}
	return &CrushEngine{BehaviorDefinedEngine: engine}
}
