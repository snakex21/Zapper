package edge

type _ICoreWebView2CapturePreviewCompletedHandlerVtbl struct {
	_IUnknownVtbl
	Invoke ComProc
}

type iCoreWebView2CapturePreviewCompletedHandler struct {
	vtbl *_ICoreWebView2CapturePreviewCompletedHandlerVtbl
	impl _ICoreWebView2CapturePreviewCompletedHandlerImpl
}

type _ICoreWebView2CapturePreviewCompletedHandlerImpl interface {
	_IUnknownImpl
	CapturePreviewCompleted(errorCode uintptr) uintptr
}

func _ICoreWebView2CapturePreviewCompletedHandlerIUnknownQueryInterface(this *iCoreWebView2CapturePreviewCompletedHandler, refiid, object uintptr) uintptr {
	return this.impl.QueryInterface(refiid, object)
}

func _ICoreWebView2CapturePreviewCompletedHandlerIUnknownAddRef(this *iCoreWebView2CapturePreviewCompletedHandler) uintptr {
	return this.impl.AddRef()
}

func _ICoreWebView2CapturePreviewCompletedHandlerIUnknownRelease(this *iCoreWebView2CapturePreviewCompletedHandler) uintptr {
	return this.impl.Release()
}

func _ICoreWebView2CapturePreviewCompletedHandlerInvoke(this *iCoreWebView2CapturePreviewCompletedHandler, errorCode uintptr) uintptr {
	return this.impl.CapturePreviewCompleted(errorCode)
}

var _ICoreWebView2CapturePreviewCompletedHandlerFn = _ICoreWebView2CapturePreviewCompletedHandlerVtbl{
	_IUnknownVtbl{
		NewComProc(_ICoreWebView2CapturePreviewCompletedHandlerIUnknownQueryInterface),
		NewComProc(_ICoreWebView2CapturePreviewCompletedHandlerIUnknownAddRef),
		NewComProc(_ICoreWebView2CapturePreviewCompletedHandlerIUnknownRelease),
	},
	NewComProc(_ICoreWebView2CapturePreviewCompletedHandlerInvoke),
}

func newICoreWebView2CapturePreviewCompletedHandler(impl _ICoreWebView2CapturePreviewCompletedHandlerImpl) *iCoreWebView2CapturePreviewCompletedHandler {
	return &iCoreWebView2CapturePreviewCompletedHandler{
		vtbl: &_ICoreWebView2CapturePreviewCompletedHandlerFn,
		impl: impl,
	}
}
