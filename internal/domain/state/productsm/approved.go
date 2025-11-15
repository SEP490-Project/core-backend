package productsm

//type ApprovedState struct{}
//
//func (a ApprovedState) Name() enum.ProductStatus {
//	return enum.ProductStatusApproved
//}
//
//func (a ApprovedState) Next(ctx *ProductContext, next ProductState) error {
//	if _, ok := a.AllowedTransitions()[next.Name()]; ok {
//		ctx.State = next
//		return nil
//	}
//	return fmt.Errorf("invalid transition: %s -> %s", a.Name(), next.Name())
//}
//
//func (a ApprovedState) AllowedTransitions() map[enum.ProductStatus]struct{} {
//	return map[enum.ProductStatus]struct{}{
//		enum.ProductStatusActived:   {},
//		enum.ProductStatusInactived: {},
//	}
//}
