package mediumpk

import(
	"testing"

	"github.com/stretchr/testify/assert"
)
var maxPending int = 64

func TestMediumpk_New(t *testing.T) {
	// new mediumpk
	mediumpk, err := New(maxPending)
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)

	// close mediumpk
	err = mediumpk.Close()
	assert.NoError(t, err)
	
}

func TestMediumpk_Store_Channel(t *testing.T) {
	// new mediumpk
	mediumpk, err := New(maxPending)
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
	chanStore :=	make([]*chan ResponseEnvelop, maxPending)

	//set slice of channel pointers
	for i, _ := range(chanStore){
		resChan := make(chan ResponseEnvelop)
		assert.Nil(t, chanStore[i])
		chanStore[i] = &resChan
		assert.NotNil(t, chanStore[i])
	}
	
	// store channel pointers
	for i, v := range(chanStore){
		index, err := mediumpk.putChannel(v)
		assert.NoError(t, err)
		assert.Equal(t, i, index)
	}
	
	// full of channel pointers.. return error
	resChan := make(chan ResponseEnvelop)
	index, err := mediumpk.putChannel(&resChan)
	assert.Error(t, err)
	assert.Equal(t, index, -1)

	// get all stored channel pointers and check whether it is right one
	for i, v := range(chanStore){
		pchan, err := mediumpk.getChannel(i)
		assert.NoError(t, err)
		assert.Equal(t, v, pchan)
	}

	// close mediumpk
	err = mediumpk.Close()
	assert.NoError(t, err)
	assert.NotNil(t, mediumpk)
}