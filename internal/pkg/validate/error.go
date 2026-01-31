package validate

type FieldsError struct {
	Fields map[string]string
}

func NewFieldsError(fields map[string]string) *FieldsError {
	return &FieldsError{
		Fields: fields,
	}
}
func (f *FieldsError) Error() string {
	return "Fields error"
}
