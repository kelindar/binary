//go:build repro

package binary

type reproUnionMapArm struct {
	Values map[string][]byte
}

type reproUnionMapContainer struct {
	Arm *reproUnionMapArm `binary:"1,union"`
}

type reproUnionMapEnvelope struct {
	Body reproUnionMapContainer
	Tail map[string][]byte
}
