package schema

// SingleAnalysisOutput is for one of the core algorithms.
type SingleAnalysisOutput struct {
	FileResults []FileResult
	*AggregateOutput
}

// CompareAnalysisOutput is for one of the core algorithms.
type CompareAnalysisOutput struct {
	FileResults   []FileResult
	FolderResults []FolderResult
}
