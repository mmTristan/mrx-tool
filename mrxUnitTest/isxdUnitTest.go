package mrxUnitTest

import (
	"fmt"
	"io"

	mxf2go "github.com/metarex-media/mxf-to-go"
	. "github.com/onsi/gomega"
)

// ISXDDoc is the document the specs for
// ISXD are found in and used for these tests.
const ISXDDoc = "RDD47:2018"

// ISXDSpecifications returns all the specifications
// associated with ISXD
func ISXDSpecifications() Specifications {
	nt := testISXDDescriptor

	gc := genericCountCheck
	tfw := testFrameWrapped

	tgp := testGenericPartition

	ts := checkStructure

	/*
		improvements get the documentation involved



		remove primers from part of it? How many partitions are there likely to be.
		leave the spec as part of the test so it doesn't get lost etc?
	*/

	return Specifications{
		Node: map[string][]*func(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test){
			mxf2go.GISXDUL[13:]: {&nt},
			// 060e2b34.02530101.0d010101.01013b00
		},
		Part: map[string][]*func(doc io.ReadSeeker, isxdDesc *PartitionNode) func(t Test){
			HeaderKey:  {&gc},
			EssenceKey: {&tfw},
			GenericKey: {&tgp},
		},
		MXF: []*func(doc io.ReadSeeker, isxdDesc *MXFNode) func(t Test){&ts},
	}

}

func testISXDDescriptor(doc io.ReadSeeker, isxdDesc *Node, primer map[string]string) func(t Test) {

	return func(t Test) {

		// rdd-47:2009/11.5.3/shall/4
		t.Test("Checking that the isxd descriptor is present in the header metadata", NewSpec(ISXDDoc, "9.2", "shall", 1),
			t.Expect(isxdDesc).ShallNot(BeNil()),
		)

		if isxdDesc != nil {
			// decode the group
			isxdDecode, err := DecodeGroupNode(doc, isxdDesc, primer)

			t.Test("Checking that the data essence coding filed is present in the isxd descriptor", NewSpec(ISXDDoc, "9.3", "shall", 1),
				t.Expect(err).Shall(BeNil()),
				t.Expect(isxdDecode["DataEssenceCoding"]).Shall(Equal(mxf2go.TAUID{
					Data1: 101591860,
					Data2: 1025,
					Data3: 261,
					Data4: mxf2go.TUInt8Array8{14, 9, 6, 6, 0, 0, 0, 0},
				})))
		}
	}
}

// check the generic body partition count and layout to the ISXD spec
func genericCountCheck(doc io.ReadSeeker, header *PartitionNode) func(t Test) {

	return func(t Test) {

		genericParts, err := header.Parent.Search("select * from partitions where type = " + GenericStreamPartition)

		t.Test("Checking there is no error getting the generic partition streams", NewSpec(ISXDDoc, "5.4", "shall", 1),
			t.Expect(err).To(BeNil()),
		)

		if len(genericParts) > 0 {
			// ibly run if there's any generic essence
			// update to a partitionsearch

			staticTracks, err := header.Search("select * from metadata where UL = " + mxf2go.GStaticTrackUL[13:])
			t.Test("Checking that a single static track is present in the header metadata", NewSpec(ISXDDoc, "5.4", "shall", 1),
				t.Expect(err).To(BeNil()),
				t.Expect(len(staticTracks)).Shall(Equal(1)),
			)

			if len(staticTracks) == 1 {
				staticTrack := staticTracks[0]

				t.Test("Checking that the static track is not nil", NewSpec(ISXDDoc, "5.4", "shall", 1),
					t.Expect(staticTrack).ShallNot(BeNil()),
				)

				sequence := staticTrack.FindUL(mxf2go.GSequenceUL[13:])
				t.Test("Checking that the static track points to a sequence", NewSpec(ISXDDoc, "5.4", "shall", 2),
					t.Expect(sequence).ToNot(BeNil()),
				)

				t.Test("Checking that the static track sequence has as many sequence children as partitions", NewSpec(ISXDDoc, "5.4", "shall", 2),
					t.Expect(len(sequence.Children)).Shall(Equal(len(genericParts))),
				)
			}
		}

	}
	// test ISXD descriptor

}

// this is a header test
func testFrameWrapped(doc io.ReadSeeker, header *PartitionNode) func(t Test) {
	return func(t Test) {

		if len(header.Essence) > 0 {

			badKeys, err := header.Search("select * from essence where UL <> " + mxf2go.FrameWrappedISXDData.UL[13:])

			t.Test("Checking that the only ISXD essence keys are found in body partitions", NewSpec(ISXDDoc, "7.5", "shall", 1),
				t.Expect(err).Shall(BeNil()),
				t.Expect(len(badKeys)).Shall(Equal(0), fmt.Sprintf("%v other essence keys found", len(badKeys))),
			)

			if len(badKeys) != 0 {

				fwPattern := header.Props.EssenceOrder
				breakPoint := 0
				// check each header against the pattern.
				for i, e := range header.Essence {
					ess := nodeToKLV(doc, e)
					if fullNameMask(ess.Key) != fwPattern[i%len(fwPattern)] {
						breakPoint = e.Key.Start
						break
					}

				}

				t.Test("Checking that the content package order are regular throughout the essence stream", NewSpec(ISXDDoc, "7.5", "shall", 1),
					t.Expect(breakPoint).Shall(Equal(0), fmt.Sprintf("irregular key found at byte offset %v", breakPoint)),
				)
			}
		}
	}
}

func testGenericPartition(doc io.ReadSeeker, header *PartitionNode) func(t Test) {
	return func(t Test) {

		headerKLV := nodeToKLV(doc, &Node{Key: header.Key, Length: header.Length, Value: header.Value})
		mp := partitionExtract(headerKLV)

		t.Test("Checking that the index byte count for the generic header is 0", NewSpec(ISXDDoc, "7.5", "shall", 1),
			t.Expect(mp.IndexByteCount).Shall(Equal(uint64(0)), "index byte count not 0"),
		)

		t.Test("Checking that the header metadata byte count for the generic header is 0", NewSpec(ISXDDoc, "7.5", "shall", 1),
			t.Expect(mp.HeaderByteCount).Shall(Equal(uint64(0)), "header metadata byte count not 0"),
		)

		t.Test("Checking that the index SID for the generic header is 0", NewSpec(ISXDDoc, "7.5", "shall", 1),
			t.Expect(mp.IndexSID).Shall(Equal(uint32(0)), "index SID not 0"),
		)

		t.Test("checking the partition key meets the expected value of "+mxf2go.GGenericStreamPartitionUL[13:], NewSpec(ISXDDoc, "7.5", "shall", 1),
			t.Expect(fullNameMask(headerKLV.Key, 5)).Shall(Equal(mxf2go.GGenericStreamPartitionUL[13:])),
		)

		// 060e2b34.0101010c.0d010509.01000000 as the value is not used in the registers (yet?)
		gpEssKey := "060e2b34.0101010c.0d010509.01000000"
		invalidKeys, err := header.Search("select * from essence where ul <> " + gpEssKey)
		// 09.01 - 1001 -little endin & 01 - makrer bit
		// can be shown as this but is not in the essence
		// 060e2b34.0101010c.0d01057f.7f000000

		t.Test("checking the essence keys all have the value of "+gpEssKey, NewSpec(ISXDDoc, "7.5", "shall", 1),
			t.Expect(err).Shall(BeNil()),
			t.Expect(len(invalidKeys)).Shall(Equal(0), fmt.Sprintf("%v other essence keys found", len(invalidKeys))),
		)
	}
}

func checkStructure(doc io.ReadSeeker, mxf *MXFNode) func(t Test) {
	return func(t Test) {

		// find the generic paritions

		genericParts, gpErr := mxf.Search("select * from partitions where type = " + GenericStreamPartition)
		// find the generic partitions positions
		GenericCountPositions := make([]int, len(genericParts))
		for i, gcp := range genericParts {
			GenericCountPositions[i] = gcp.PartitionPos
		}

		endPos := len(mxf.Partitions)
		footerParts, footErr := mxf.Search("select * from partitions where type = " + FooterPartition)
		if len(footerParts) != 0 {
			endPos--
		}

		ripParts, ripErr := mxf.Search("select * from partitions where type = " + RIPPartition)
		if len(ripParts) != 0 {
			endPos--
		}

		expectedParts := make([]int, len(GenericCountPositions))
		for j := range expectedParts {
			expectedParts[j] = endPos - len(expectedParts) + j
		}
		t.Test("Checking that the generic partition positions match the expected positions at the end of the file", NewSpec(ISXDDoc, "5.4", "shall", 3),
			t.Expect(gpErr).To(BeNil()),
			t.Expect(footErr).To(BeNil()),
			t.Expect(ripErr).To(BeNil()),
			t.Expect(expectedParts).Shall(Equal(GenericCountPositions)),
		)
	}
}
