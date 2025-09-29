package uploadloader

// Option configures UploadLoader
type Option func(*UploadLoader)

// WithMaxAllowedSize sets maximum allowed size for uploaded images
func WithMaxAllowedSize(size int) Option {
	return func(u *UploadLoader) {
		u.MaxAllowedSize = size
	}
}

// WithAccept sets accepted Content-Type for uploads
func WithAccept(accept string) Option {
	return func(u *UploadLoader) {
		u.Accept = accept
	}
}

// WithFormFieldName sets the form field name for multipart uploads
func WithFormFieldName(fieldName string) Option {
	return func(u *UploadLoader) {
		u.FormFieldName = fieldName
	}
}
