// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package s256

import (
	"fmt"
	"github.com/magiconair/properties/assert"
	"sync"
	"testing"
	"time"
)

func TestVRF(t *testing.T) {
	k, pk := GenerateKey()

	m1 := []byte("data1")
	m2 := []byte("data2")
	m3 := []byte("data2")
	index1, proof1 := k.Evaluate(m1)
	index2, proof2 := k.Evaluate(m2)
	index3, proof3 := k.Evaluate(m3)
	for _, tc := range []struct {
		m     []byte
		index [32]byte
		proof []byte
		err   error
	}{
		{m1, index1, proof1, nil},
		{m2, index2, proof2, nil},
		{m3, index3, proof3, nil},
		{m3, index3, proof2, nil},
		{m3, index3, proof1, ErrInvalidVRF},
	} {
		index, err := pk.ProofToHash(tc.m, tc.proof)
		if got, want := err, tc.err; got != want {
			t.Errorf("ProofToHash(%s, %x): %v, want %v", tc.m, tc.proof, got, want)
		}
		if err != nil {
			continue
		}
		if got, want := index, tc.index; got != want {
			t.Errorf("ProofToInex(%s, %x): %x, want %x", tc.m, tc.proof, got, want)
		}
	}
}

func TestVRF0(t *testing.T) {
	start := time.Now()
	var wg sync.WaitGroup
	for i:=0;i<20000;i++{
		go func() {
			wg.Add(1)
			defer wg.Done()
			k, pk := GenerateKey()
			m := []byte("data1")
			output, proof := k.Evaluate(m)
			output1, err := pk.ProofToHash(m, proof)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, output, output1, "not equal")
		}()
	}
	wg.Wait()
	end := time.Now()
	fmt.Println(end.Sub(start))
}
